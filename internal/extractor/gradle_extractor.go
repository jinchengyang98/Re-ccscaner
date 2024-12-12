package extractor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yourusername/ccscanner/pkg/models"
)

// GradleExtractor 实现了从 Gradle 构建文件中提取依赖信息的提取器
type GradleExtractor struct {
	BaseExtractor
}

// NewGradleExtractor 创建一个新的 Gradle 提取器实例
func NewGradleExtractor() *GradleExtractor {
	return &GradleExtractor{
		BaseExtractor: BaseExtractor{
			name:        "Gradle",
			filePattern: regexp.MustCompile(`build\.gradle|build\.gradle\.kts|settings\.gradle|settings\.gradle\.kts`),
		},
	}
}

// Extract 从 Gradle 构建文件中提取依赖信息
// 参数:
//   - projectPath: 项目根目录路径
//   - filePath: Gradle 构建文件路径
//
// 返回:
//   - []models.Dependency: 提取到的依赖列表
//   - error: 错误信息
func (e *GradleExtractor) Extract(projectPath string, filePath string) ([]models.Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	var dependencies []models.Dependency
	scanner := bufio.NewScanner(file)

	// 用于匹配各种 Gradle 依赖声明的正则表达式
	nativeDependencyRegex := regexp.MustCompile(`native(?:Lib|Implementation|Api)\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	pluginDependencyRegex := regexp.MustCompile(`id\s*['"]([^'"]+)['"]\s*version\s*['"]([^'"]+)['"]`)
	projectDependencyRegex := regexp.MustCompile(`project\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	includeDirRegex := regexp.MustCompile(`cppCompiler\.includeDirs\.from\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	var inNativeBlock bool
	var currentComponent string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// 检查是否进入 native 组件块
		if strings.Contains(line, "components.native") {
			inNativeBlock = true
			if matches := regexp.MustCompile(`components\.native\s*\{\s*([^}]+)\s*\}`).FindStringSubmatch(line); len(matches) > 1 {
				currentComponent = matches[1]
			}
			continue
		}

		// 检查是否退出 native 块
		if inNativeBlock && line == "}" {
			inNativeBlock = false
			currentComponent = ""
			continue
		}

		// 提取 native 依赖
		if matches := nativeDependencyRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "gradle_native",
				FilePath: filePath,
				Line:     lineNum,
				Parent:   currentComponent,
			}
			dependencies = append(dependencies, dep)
		}

		// 提取插件依赖
		if matches := pluginDependencyRegex.FindStringSubmatch(line); len(matches) > 2 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "gradle_plugin",
				FilePath: filePath,
				Line:     lineNum,
				Version:  matches[2],
			}
			dependencies = append(dependencies, dep)
		}

		// 提取项目依赖
		if matches := projectDependencyRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "gradle_project",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
		}

		// 提取包含目录
		if matches := includeDirRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "gradle_include_dir",
				FilePath: filePath,
				Line:     lineNum,
				Parent:   currentComponent,
			}
			dependencies = append(dependencies, dep)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %v", filePath, err)
	}

	// 如果是 settings.gradle,还需要处理子项目
	if isSettingsFile(filePath) {
		subprojects, err := e.extractSubprojects(filePath)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, subprojects...)
	}

	return dependencies, nil
}

// isSettingsFile 检查是否是 settings.gradle 文件
func isSettingsFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	return strings.HasPrefix(fileName, "settings.gradle")
}

// extractSubprojects 从 settings.gradle 文件中提取子项目信息
func (e *GradleExtractor) extractSubprojects(filePath string) ([]models.Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open settings file %s: %v", filePath, err)
	}
	defer file.Close()

	var dependencies []models.Dependency
	scanner := bufio.NewScanner(file)
	includeRegex := regexp.MustCompile(`include\s*['"]([^'"]+)['"]`)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if matches := includeRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "gradle_subproject",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning settings file %s: %v", filePath, err)
	}

	return dependencies, nil
}

// GetName 返回提取器的名称
func (e *GradleExtractor) GetName() string {
	return e.name
}

// IsApplicable 检查提取器是否适用于指定的文件
func (e *GradleExtractor) IsApplicable(filePath string) bool {
	fileName := filepath.Base(filePath)
	return e.filePattern.MatchString(fileName)
}

// GetPriority 返回提取器的优先级
func (e *GradleExtractor) GetPriority() int {
	return 100
}

// GetFilePattern 返回提取器的文件模式
func (e *GradleExtractor) GetFilePattern() *regexp.Regexp {
	return e.filePattern
}

// String 返回提取器的字符串表示
func (e *GradleExtractor) String() string {
	return fmt.Sprintf("GradleExtractor{name: %s, pattern: %s}", e.name, e.filePattern.String())
}

// 注意事项:
// 1. Gradle 构建文件使用 Groovy 或 Kotlin DSL,需要处理两种语法
// 2. 需要处理多项目构建的情况
// 3. 需要处理插件依赖和项目依赖
// 4. 可能需要处理自定义的依赖配置
// 5. 建议添加对 buildSrc 目录的支持

// 使用示例:
// extractor := NewGradleExtractor()
// deps, err := extractor.Extract("/path/to/project", "/path/to/project/build.gradle")
// if err != nil {
//     log.Fatal(err)
// }
// for _, dep := range deps {
//     fmt.Printf("Found dependency: %s (%s)\n", dep.Name, dep.Type)
// } 