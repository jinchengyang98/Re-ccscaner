package extractor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/ccscanner/pkg/models"
)

// CocoaPodsExtractor 实现了iOS/macOS CocoaPods项目的依赖提取器
type CocoaPodsExtractor struct {
	BaseExtractor
}

// PodfileLock 表示Podfile.lock的结构
type PodfileLock struct {
	PODS []struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Dependencies []string `json:"dependencies,omitempty"`
	} `json:"PODS"`
	DEPENDENCIES []string `json:"DEPENDENCIES"`
	SPEC_REPOS  map[string][]string `json:"SPEC REPOS"`
	EXTERNAL_SOURCES map[string]struct {
		Path   string `json:"path,omitempty"`
		Git    string `json:"git,omitempty"`
		Tag    string `json:"tag,omitempty"`
		Branch string `json:"branch,omitempty"`
		Commit string `json:"commit,omitempty"`
	} `json:"EXTERNAL SOURCES"`
	CHECKOUT_OPTIONS map[string]struct {
		Git     string `json:"git"`
		Commit  string `json:"commit"`
	} `json:"CHECKOUT OPTIONS"`
	SPEC_CHECKSUMS map[string]string `json:"SPEC CHECKSUMS"`
	PODFILE_CHECKSUM string `json:"PODFILE CHECKSUM"`
}

// NewCocoaPodsExtractor 创建一个新的CocoaPods提取器实例
func NewCocoaPodsExtractor() *CocoaPodsExtractor {
	return &CocoaPodsExtractor{
		BaseExtractor: BaseExtractor{
			name: "cocoapods",
			patterns: []string{
				"Podfile",
				"Podfile.lock",
			},
		},
	}
}

// Extract 从Podfile和Podfile.lock中提取依赖信息
func (e *CocoaPodsExtractor) Extract(path string) ([]*models.Dependency, error) {
	// 检查Podfile是否存在
	podfilePath := filepath.Join(path, "Podfile")
	if _, err := os.Stat(podfilePath); err != nil {
		return nil, fmt.Errorf("Podfile not found: %v", err)
	}

	// 检查Podfile.lock是否存在
	lockPath := filepath.Join(path, "Podfile.lock")
	if _, err := os.Stat(lockPath); err != nil {
		return nil, fmt.Errorf("Podfile.lock not found: %v", err)
	}

	// 读取并解析Podfile.lock
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Podfile.lock: %v", err)
	}

	var lock PodfileLock
	if err := json.Unmarshal(lockData, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse Podfile.lock: %v", err)
	}

	// 提取依赖信息
	var dependencies []*models.Dependency

	// 处理每个pod
	for _, pod := range lock.PODS {
		// 解析pod名称和版本
		name := pod.Name
		version := pod.Version

		// 如果名称包含版本号,提取实际的名称
		if strings.Contains(name, " (") {
			parts := strings.Split(name, " (")
			name = parts[0]
			version = strings.TrimSuffix(parts[1], ")")
		}

		dependency := &models.Dependency{
			Name:    name,
			Version: version,
			Type:    "cocoapods",
		}

		// 添加依赖关系信息
		if len(pod.Dependencies) > 0 {
			dependency.Metadata = map[string]interface{}{
				"dependencies": pod.Dependencies,
			}
		}

		// 检查是否是外部源
		if extSource, ok := lock.EXTERNAL_SOURCES[name]; ok {
			if extSource.Git != "" {
				dependency.Source = extSource.Git
				if extSource.Tag != "" {
					dependency.Version = fmt.Sprintf("tag=%s", extSource.Tag)
				} else if extSource.Branch != "" {
					dependency.Version = fmt.Sprintf("branch=%s", extSource.Branch)
				} else if extSource.Commit != "" {
					dependency.Version = fmt.Sprintf("commit=%s", extSource.Commit)
				}
			} else if extSource.Path != "" {
				dependency.Source = fmt.Sprintf("path=%s", extSource.Path)
			}
		}

		// 检查是否有checkout选项
		if checkout, ok := lock.CHECKOUT_OPTIONS[name]; ok {
			dependency.Source = checkout.Git
			dependency.Version = fmt.Sprintf("commit=%s", checkout.Commit)
		}

		// 添加checksum信息
		if checksum, ok := lock.SPEC_CHECKSUMS[name]; ok {
			if dependency.Metadata == nil {
				dependency.Metadata = make(map[string]interface{})
			}
			dependency.Metadata["checksum"] = checksum
		}

		dependencies = append(dependencies, dependency)
	}

	// 标记开发依赖
	// 注意: CocoaPods没有明确的开发依赖概念,我们可以通过配置或命名约定来识别
	for _, dep := range dependencies {
		if strings.HasSuffix(dep.Name, "Tests") || 
		   strings.HasSuffix(dep.Name, "Testing") ||
		   strings.HasSuffix(dep.Name, "Mock") ||
		   strings.HasSuffix(dep.Name, "Spec") {
			dep.Scope = "dev"
		}
	}

	return dependencies, nil
}

// 注意事项:
// 1. CocoaPods使用YAML格式的Podfile.lock,这里为了简化使用了JSON格式
// 2. 实际使用时需要添加YAML解析支持
// 3. 可以通过分析Podfile获取更多信息,如target特定的依赖
// 4. 可以添加对subspecs的支持
// 5. 可以添加对私有pod源的支持 