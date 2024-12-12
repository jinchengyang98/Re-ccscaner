package extractor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/ccscanner/pkg/models"
)

// CargoExtractor 实现了Rust Cargo项目的依赖提取器
type CargoExtractor struct {
	BaseExtractor
}

// CargoManifest 表示Cargo.toml的结构
type CargoManifest struct {
	Package  CargoPackage         `json:"package"`
	Dependencies map[string]CargoDependency `json:"dependencies"`
	DevDependencies map[string]CargoDependency `json:"dev-dependencies"`
}

type CargoPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type CargoDependency struct {
	Version  string `json:"version,omitempty"`
	Git      string `json:"git,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Rev      string `json:"rev,omitempty"`
	Features []string `json:"features,omitempty"`
}

// NewCargoExtractor 创建一个新的Cargo提取器实例
func NewCargoExtractor() *CargoExtractor {
	return &CargoExtractor{
		BaseExtractor: BaseExtractor{
			name: "cargo",
			patterns: []string{
				"Cargo.toml",
				"Cargo.lock",
			},
		},
	}
}

// Extract 从Cargo.toml和Cargo.lock中提取依赖信息
func (e *CargoExtractor) Extract(path string) ([]*models.Dependency, error) {
	// 检查Cargo.toml是否存在
	manifestPath := filepath.Join(path, "Cargo.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, fmt.Errorf("Cargo.toml not found: %v", err)
	}

	// 读取并解析Cargo.toml
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Cargo.toml: %v", err)
	}

	var manifest CargoManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse Cargo.toml: %v", err)
	}

	// 提取依赖信息
	var dependencies []*models.Dependency

	// 处理正常依赖
	for name, dep := range manifest.Dependencies {
		dependency := &models.Dependency{
			Name:    name,
			Version: dep.Version,
			Type:    "cargo",
		}

		// 处理Git依赖
		if dep.Git != "" {
			dependency.Source = dep.Git
			if dep.Branch != "" {
				dependency.Version = fmt.Sprintf("branch=%s", dep.Branch)
			} else if dep.Rev != "" {
				dependency.Version = fmt.Sprintf("rev=%s", dep.Rev)
			}
		}

		// 添加特性信息
		if len(dep.Features) > 0 {
			dependency.Metadata = map[string]interface{}{
				"features": dep.Features,
			}
		}

		dependencies = append(dependencies, dependency)
	}

	// 处理开发依赖
	for name, dep := range manifest.DevDependencies {
		dependency := &models.Dependency{
			Name:    name,
			Version: dep.Version,
			Type:    "cargo",
			Scope:   "dev",
		}

		// 处理Git依赖
		if dep.Git != "" {
			dependency.Source = dep.Git
			if dep.Branch != "" {
				dependency.Version = fmt.Sprintf("branch=%s", dep.Branch)
			} else if dep.Rev != "" {
				dependency.Version = fmt.Sprintf("rev=%s", dep.Rev)
			}
		}

		// 添加特性信息
		if len(dep.Features) > 0 {
			dependency.Metadata = map[string]interface{}{
				"features": dep.Features,
			}
		}

		dependencies = append(dependencies, dependency)
	}

	// 尝试从Cargo.lock获取更精确的版本信息
	lockPath := filepath.Join(path, "Cargo.lock")
	if _, err := os.Stat(lockPath); err == nil {
		lockData, err := os.ReadFile(lockPath)
		if err == nil {
			var lockFile struct {
				Package []struct {
					Name    string `json:"name"`
					Version string `json:"version"`
					Source  string `json:"source,omitempty"`
				} `json:"package"`
			}
			if err := json.Unmarshal(lockData, &lockFile); err == nil {
				// 更新依赖的精确版本
				for _, dep := range dependencies {
					for _, pkg := range lockFile.Package {
						if strings.EqualFold(dep.Name, pkg.Name) {
							dep.Version = pkg.Version
							if pkg.Source != "" {
								dep.Source = pkg.Source
							}
							break
						}
					}
				}
			}
		}
	}

	return dependencies, nil
} 