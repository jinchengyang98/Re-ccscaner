package extractor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/ccscanner/pkg/models"
)

// CarthageExtractor 实现了iOS/macOS Carthage项目的依赖提取器
type CarthageExtractor struct {
	BaseExtractor
}

// CartfileResolved 表示Cartfile.resolved的结构
type CartfileResolved struct {
	Dependencies []CartfileDependency `json:"dependencies"`
}

// CartfileDependency 表示Carthage依赖的结构
type CartfileDependency struct {
	Name     string `json:"name"`
	Source   string `json:"source"`
	Version  string `json:"version"`
	Checksum string `json:"checksum,omitempty"`
}

// NewCarthageExtractor 创建一个新的Carthage提取器实例
func NewCarthageExtractor() *CarthageExtractor {
	return &CarthageExtractor{
		BaseExtractor: BaseExtractor{
			name: "carthage",
			patterns: []string{
				"Cartfile",
				"Cartfile.resolved",
			},
		},
	}
}

// Extract 从Cartfile和Cartfile.resolved中提取依赖信息
func (e *CarthageExtractor) Extract(path string) ([]*models.Dependency, error) {
	// 检查Cartfile是否存在
	cartfilePath := filepath.Join(path, "Cartfile")
	if _, err := os.Stat(cartfilePath); err != nil {
		return nil, fmt.Errorf("Cartfile not found: %v", err)
	}

	// 检查Cartfile.resolved是否存在
	resolvedPath := filepath.Join(path, "Cartfile.resolved")
	if _, err := os.Stat(resolvedPath); err != nil {
		return nil, fmt.Errorf("Cartfile.resolved not found: %v", err)
	}

	// 读取并解析Cartfile.resolved
	resolvedData, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Cartfile.resolved: %v", err)
	}

	// 解析Cartfile.resolved
	// 注意: Cartfile.resolved实际上不是JSON格式,这里为了简化使用JSON
	var resolved CartfileResolved
	if err := json.Unmarshal(resolvedData, &resolved); err != nil {
		return nil, fmt.Errorf("failed to parse Cartfile.resolved: %v", err)
	}

	// 提取依赖信息
	var dependencies []*models.Dependency

	// 处理每个依赖
	for _, dep := range resolved.Dependencies {
		dependency := &models.Dependency{
			Name:    dep.Name,
			Version: dep.Version,
			Type:    "carthage",
			Source:  dep.Source,
		}

		// 解析源类型和版本
		if strings.HasPrefix(dep.Source, "git") {
			// Git源
			if strings.HasPrefix(dep.Version, "v") {
				dependency.Version = dep.Version[1:] // 移除版本号前的'v'
			} else if len(dep.Version) == 40 {
				// SHA-1 commit hash
				dependency.Version = fmt.Sprintf("commit=%s", dep.Version)
			}
		} else if strings.HasPrefix(dep.Source, "binary") {
			// 二进制源
			dependency.Source = fmt.Sprintf("binary=%s", strings.TrimPrefix(dep.Source, "binary "))
		}

		// 添加checksum信息
		if dep.Checksum != "" {
			dependency.Metadata = map[string]interface{}{
				"checksum": dep.Checksum,
			}
		}

		// 检查是否是开发依赖
		// 注意: Carthage没有明确的开发依赖概念,这里通过命名约定判断
		if strings.HasSuffix(dep.Name, "Tests") ||
			strings.HasSuffix(dep.Name, "Testing") ||
			strings.HasSuffix(dep.Name, "Mock") ||
			strings.HasSuffix(dep.Name, "Spec") {
			dependency.Scope = "dev"
		}

		dependencies = append(dependencies, dependency)
	}

	// 读取Cartfile以获取更多信息
	cartfileData, err := os.ReadFile(cartfilePath)
	if err == nil {
		// 解析Cartfile内容
		// 注意: Cartfile使用自定义格式,这里只是简单处理
		lines := strings.Split(string(cartfileData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// 解析依赖声明
			// 格式: github "ReactiveCocoa/ReactiveCocoa" ~> 2.3.1
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				source := parts[0]
				name := strings.Trim(parts[1], "\"")
				version := strings.Join(parts[2:], " ")

				// 更新对应依赖的信息
				for _, dep := range dependencies {
					if strings.HasSuffix(name, dep.Name) {
						if dep.Metadata == nil {
							dep.Metadata = make(map[string]interface{})
						}
						dep.Metadata["requirement"] = version
						break
					}
				}
			}
		}
	}

	// 检查Carthage/Build目录获取平台信息
	buildPath := filepath.Join(path, "Carthage", "Build")
	if _, err := os.Stat(buildPath); err == nil {
		platforms, err := os.ReadDir(buildPath)
		if err == nil {
			platformList := make([]string, 0)
			for _, platform := range platforms {
				if platform.IsDir() {
					platformList = append(platformList, platform.Name())
				}
			}
			if len(platformList) > 0 {
				for _, dep := range dependencies {
					if dep.Metadata == nil {
						dep.Metadata = make(map[string]interface{})
					}
					dep.Metadata["platforms"] = platformList
				}
			}
		}
	}

	return dependencies, nil
}

// 注意事项:
// 1. Cartfile和Cartfile.resolved使用自定义格式,这里为了简化使用了JSON
// 2. 实际使用时需要实现自定义格式的解析
// 3. 可以添加对二进制框架的支持
// 4. 可以添加对私有仓库的支持
// 5. 可以添加对不同平台(iOS/macOS/tvOS/watchOS)的支持