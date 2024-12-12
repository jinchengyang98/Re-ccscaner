package extractor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/ccscanner/pkg/models"
)

// PoetryExtractor 实现了Python Poetry项目的依赖提取器
type PoetryExtractor struct {
	BaseExtractor
}

// PoetryConfig 表示pyproject.toml的结构
type PoetryConfig struct {
	Tool struct {
		Poetry struct {
			Name         string                     `json:"name"`
			Version      string                     `json:"version"`
			Description  string                     `json:"description"`
			Dependencies map[string]interface{}     `json:"dependencies"`
			DevDependencies map[string]interface{} `json:"dev-dependencies"`
		} `json:"poetry"`
	} `json:"tool"`
}

// NewPoetryExtractor 创建一个新的Poetry提取器实例
func NewPoetryExtractor() *PoetryExtractor {
	return &PoetryExtractor{
		BaseExtractor: BaseExtractor{
			name: "poetry",
			patterns: []string{
				"pyproject.toml",
				"poetry.lock",
			},
		},
	}
}

// parseDependencyVersion 解析依赖版本信息
func parseDependencyVersion(dep interface{}) (string, string, map[string]interface{}, error) {
	var version, source string
	var metadata map[string]interface{}

	switch v := dep.(type) {
	case string:
		version = v
	case map[string]interface{}:
		if ver, ok := v["version"].(string); ok {
			version = ver
		}
		if src, ok := v["source"].(string); ok {
			source = src
		}
		if extras, ok := v["extras"].([]interface{}); ok {
			metadata = map[string]interface{}{
				"extras": extras,
			}
		}
		if git, ok := v["git"].(string); ok {
			source = git
			if rev, ok := v["rev"].(string); ok {
				version = fmt.Sprintf("rev=%s", rev)
			} else if branch, ok := v["branch"].(string); ok {
				version = fmt.Sprintf("branch=%s", branch)
			} else if tag, ok := v["tag"].(string); ok {
				version = fmt.Sprintf("tag=%s", tag)
			}
		}
	default:
		return "", "", nil, fmt.Errorf("unsupported dependency format: %v", dep)
	}

	return version, source, metadata, nil
}

// Extract 从pyproject.toml和poetry.lock中提取依赖信息
func (e *PoetryExtractor) Extract(path string) ([]*models.Dependency, error) {
	// 检查pyproject.toml是否存在
	configPath := filepath.Join(path, "pyproject.toml")
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("pyproject.toml not found: %v", err)
	}

	// 读取并解析pyproject.toml
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pyproject.toml: %v", err)
	}

	var config PoetryConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse pyproject.toml: %v", err)
	}

	// 提取依赖信息
	var dependencies []*models.Dependency

	// 处理正常依赖
	for name, dep := range config.Tool.Poetry.Dependencies {
		version, source, metadata, err := parseDependencyVersion(dep)
		if err != nil {
			return nil, err
		}

		dependency := &models.Dependency{
			Name:     name,
			Version:  version,
			Type:     "poetry",
			Source:   source,
			Metadata: metadata,
		}
		dependencies = append(dependencies, dependency)
	}

	// 处理开发依赖
	for name, dep := range config.Tool.Poetry.DevDependencies {
		version, source, metadata, err := parseDependencyVersion(dep)
		if err != nil {
			return nil, err
		}

		dependency := &models.Dependency{
			Name:     name,
			Version:  version,
			Type:     "poetry",
			Source:   source,
			Scope:    "dev",
			Metadata: metadata,
		}
		dependencies = append(dependencies, dependency)
	}

	// 尝试从poetry.lock获取更精确的版本信息
	lockPath := filepath.Join(path, "poetry.lock")
	if _, err := os.Stat(lockPath); err == nil {
		lockData, err := os.ReadFile(lockPath)
		if err == nil {
			var lockFile struct {
				Package []struct {
					Name     string   `json:"name"`
					Version  string   `json:"version"`
					Category string   `json:"category"`
					Source   struct {
						Type string `json:"type"`
						URL  string `json:"url"`
					} `json:"source"`
					Extras []string `json:"extras,omitempty"`
				} `json:"package"`
			}
			if err := json.Unmarshal(lockData, &lockFile); err == nil {
				// 更新依赖的精确版本
				for _, dep := range dependencies {
					for _, pkg := range lockFile.Package {
						if strings.EqualFold(dep.Name, pkg.Name) {
							dep.Version = pkg.Version
							if pkg.Source.Type != "" && pkg.Source.URL != "" {
								dep.Source = fmt.Sprintf("%s+%s", pkg.Source.Type, pkg.Source.URL)
							}
							if len(pkg.Extras) > 0 {
								if dep.Metadata == nil {
									dep.Metadata = make(map[string]interface{})
								}
								dep.Metadata["extras"] = pkg.Extras
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