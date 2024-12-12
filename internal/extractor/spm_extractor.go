package extractor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/ccscanner/pkg/models"
)

// SPMExtractor 实现了Swift Package Manager项目的依赖提取器
type SPMExtractor struct {
	BaseExtractor
}

// PackageConfig 表示Package.swift的结构
type PackageConfig struct {
	Name         string `json:"name"`
	Platforms    []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"platforms"`
	Products []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Targets []string `json:"targets"`
	} `json:"products"`
	Dependencies []struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Version  struct {
			LowerBound string `json:"lowerBound,omitempty"`
			UpperBound string `json:"upperBound,omitempty"`
			Exact     string `json:"exact,omitempty"`
			Branch    string `json:"branch,omitempty"`
			Revision  string `json:"revision,omitempty"`
		} `json:"version"`
		Requirement string `json:"requirement,omitempty"`
	} `json:"dependencies"`
	Targets []struct {
		Name         string   `json:"name"`
		Type         string   `json:"type"`
		Dependencies []string `json:"dependencies"`
		IsTest      bool     `json:"isTest"`
	} `json:"targets"`
}

// PackageResolved 表示Package.resolved的结构
type PackageResolved struct {
	Version int `json:"version"`
	Object struct {
		Pins []struct {
			Identity string `json:"identity"`
			Location string `json:"location"`
			State struct {
				Version     string `json:"version,omitempty"`
				Branch     string `json:"branch,omitempty"`
				Revision   string `json:"revision,omitempty"`
				CheckedOut bool   `json:"checkedOut,omitempty"`
			} `json:"state"`
		} `json:"pins"`
	} `json:"object"`
}

// NewSPMExtractor 创建一个新的Swift Package Manager提取器实例
func NewSPMExtractor() *SPMExtractor {
	return &SPMExtractor{
		BaseExtractor: BaseExtractor{
			name: "spm",
			patterns: []string{
				"Package.swift",
				"Package.resolved",
			},
		},
	}
}

// Extract 从Package.swift和Package.resolved中提取依赖信息
func (e *SPMExtractor) Extract(path string) ([]*models.Dependency, error) {
	// 检查Package.swift是否存在
	packagePath := filepath.Join(path, "Package.swift")
	if _, err := os.Stat(packagePath); err != nil {
		return nil, fmt.Errorf("Package.swift not found: %v", err)
	}

	// 读取Package.swift
	// 注意: 实际的Package.swift是Swift代码,这里为了简化使用JSON格式
	packageData, err := os.ReadFile(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Package.swift: %v", err)
	}

	var config PackageConfig
	if err := json.Unmarshal(packageData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Package.swift: %v", err)
	}

	// 提取依赖信息
	var dependencies []*models.Dependency

	// 处理每个依赖
	for _, dep := range config.Dependencies {
		dependency := &models.Dependency{
			Name:    dep.Name,
			Type:    "spm",
			Source:  dep.URL,
		}

		// 设置版本信息
		if dep.Version.Exact != "" {
			dependency.Version = dep.Version.Exact
		} else if dep.Version.Branch != "" {
			dependency.Version = fmt.Sprintf("branch=%s", dep.Version.Branch)
		} else if dep.Version.Revision != "" {
			dependency.Version = fmt.Sprintf("commit=%s", dep.Version.Revision)
		} else if dep.Version.LowerBound != "" {
			if dep.Version.UpperBound != "" {
				dependency.Version = fmt.Sprintf("%s...%s", dep.Version.LowerBound, dep.Version.UpperBound)
			} else {
				dependency.Version = fmt.Sprintf(">=%s", dep.Version.LowerBound)
			}
		}

		// 添加版本要求信息
		if dep.Requirement != "" {
			if dependency.Metadata == nil {
				dependency.Metadata = make(map[string]interface{})
			}
			dependency.Metadata["requirement"] = dep.Requirement
		}

		dependencies = append(dependencies, dependency)
	}

	// 尝试从Package.resolved获取更精确的版本信息
	resolvedPath := filepath.Join(path, "Package.resolved")
	if _, err := os.Stat(resolvedPath); err == nil {
		resolvedData, err := os.ReadFile(resolvedPath)
		if err == nil {
			var resolved PackageResolved
			if err := json.Unmarshal(resolvedData, &resolved); err == nil {
				// 更新依赖的精确版本
				for _, dep := range dependencies {
					for _, pin := range resolved.Object.Pins {
						if strings.EqualFold(dep.Name, pin.Identity) {
							if pin.State.Version != "" {
								dep.Version = pin.State.Version
							} else if pin.State.Branch != "" {
								dep.Version = fmt.Sprintf("branch=%s", pin.State.Branch)
							} else if pin.State.Revision != "" {
								dep.Version = fmt.Sprintf("commit=%s", pin.State.Revision)
							}
							break
						}
					}
				}
			}
		}
	}

	// 标记测试依赖
	for _, target := range config.Targets {
		if target.IsTest {
			for _, depName := range target.Dependencies {
				for _, dep := range dependencies {
					if strings.HasSuffix(depName, dep.Name) {
						dep.Scope = "dev"
						break
					}
				}
			}
		}
	}

	// 添加平台信息
	if len(config.Platforms) > 0 {
		platforms := make([]string, 0, len(config.Platforms))
		for _, platform := range config.Platforms {
			platforms = append(platforms, fmt.Sprintf("%s %s", platform.Name, platform.Version))
		}
		for _, dep := range dependencies {
			if dep.Metadata == nil {
				dep.Metadata = make(map[string]interface{})
			}
			dep.Metadata["platforms"] = platforms
		}
	}

	return dependencies, nil
}

// 注意事项:
// 1. Package.swift实际上是Swift代码,需要使用sourcekit-lsp或其他工具解析
// 2. Package.resolved的格式可能随Swift版本变化
// 3. 可以添加对本地包的支持
// 4. 可以添加对条件依赖的支持
// 5. 可以添加对插件的支持 