package extractor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/ccscanner/pkg/models"
)

// ComposerExtractor 实现了PHP Composer项目的依赖提取器
type ComposerExtractor struct {
	BaseExtractor
}

// ComposerConfig 表示composer.json的结构
type ComposerConfig struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type"`
	Require      map[string]string      `json:"require"`
	RequireDev   map[string]string      `json:"require-dev"`
	Repositories []ComposerRepository   `json:"repositories"`
}

// ComposerRepository 表示Composer仓库配置
type ComposerRepository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// NewComposerExtractor 创建一个新的Composer提取器实例
func NewComposerExtractor() *ComposerExtractor {
	return &ComposerExtractor{
		BaseExtractor: BaseExtractor{
			name: "composer",
			patterns: []string{
				"composer.json",
				"composer.lock",
			},
		},
	}
}

// Extract 从composer.json和composer.lock中提取依赖信息
func (e *ComposerExtractor) Extract(path string) ([]*models.Dependency, error) {
	// 检查composer.json是否存在
	configPath := filepath.Join(path, "composer.json")
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("composer.json not found: %v", err)
	}

	// 读取并解析composer.json
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read composer.json: %v", err)
	}

	var config ComposerConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse composer.json: %v", err)
	}

	// 提取依赖信息
	var dependencies []*models.Dependency

	// 处理正常依赖
	for name, version := range config.Require {
		// 跳过PHP版本要求
		if name == "php" || strings.HasPrefix(name, "ext-") {
			continue
		}

		dependency := &models.Dependency{
			Name:    name,
			Version: version,
			Type:    "composer",
		}
		dependencies = append(dependencies, dependency)
	}

	// 处理开发依赖
	for name, version := range config.RequireDev {
		// 跳过PHP版本要求
		if name == "php" || strings.HasPrefix(name, "ext-") {
			continue
		}

		dependency := &models.Dependency{
			Name:    name,
			Version: version,
			Type:    "composer",
			Scope:   "dev",
		}
		dependencies = append(dependencies, dependency)
	}

	// 尝试从composer.lock获取更精确的版本信息
	lockPath := filepath.Join(path, "composer.lock")
	if _, err := os.Stat(lockPath); err == nil {
		lockData, err := os.ReadFile(lockPath)
		if err == nil {
			var lockFile struct {
				Packages    []ComposerPackage `json:"packages"`
				PackagesDev []ComposerPackage `json:"packages-dev"`
			}
			if err := json.Unmarshal(lockData, &lockFile); err == nil {
				// 更新正常依赖的精确版本
				for _, pkg := range lockFile.Packages {
					for _, dep := range dependencies {
						if dep.Name == pkg.Name && dep.Scope == "" {
							dep.Version = pkg.Version
							if pkg.Source.Type != "" && pkg.Source.URL != "" {
								dep.Source = fmt.Sprintf("%s+%s", pkg.Source.Type, pkg.Source.URL)
							}
							if len(pkg.Require) > 0 {
								dep.Metadata = map[string]interface{}{
									"require": pkg.Require,
								}
							}
							break
						}
					}
				}

				// 更新开发依赖的精确版本
				for _, pkg := range lockFile.PackagesDev {
					for _, dep := range dependencies {
						if dep.Name == pkg.Name && dep.Scope == "dev" {
							dep.Version = pkg.Version
							if pkg.Source.Type != "" && pkg.Source.URL != "" {
								dep.Source = fmt.Sprintf("%s+%s", pkg.Source.Type, pkg.Source.URL)
							}
							if len(pkg.Require) > 0 {
								dep.Metadata = map[string]interface{}{
									"require": pkg.Require,
								}
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

// ComposerPackage 表示composer.lock中的包信息
type ComposerPackage struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Source   ComposerSource    `json:"source"`
	Dist     ComposerDist      `json:"dist"`
	Require  map[string]string `json:"require"`
	Type     string            `json:"type"`
	Time     string            `json:"time"`
}

// ComposerSource 表示包的源代码信息
type ComposerSource struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
}

// ComposerDist 表示包的分发信息
type ComposerDist struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
	Shasum    string `json:"shasum"`
} 