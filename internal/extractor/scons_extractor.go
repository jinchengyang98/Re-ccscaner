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

// SconsExtractor 实现了从 SCons 构建文件中提取依赖信息的提取器
type SconsExtractor struct {
	BaseExtractor
}

// NewSconsExtractor 创建一个新的 SCons 提取器实例
func NewSconsExtractor() *SconsExtractor {
	return &SconsExtractor{
		BaseExtractor: BaseExtractor{
			name:        "SCons",
			filePattern: regexp.MustCompile(`SConstruct|SConscript|.*\.scons$`),
		},
	}
}

// Extract 从 SCons 构建文件中提取依赖信息
// 参数:
//   - projectPath: 项目根目录路径
//   - filePath: SCons 构建文件路径
//
// 返回:
//   - []models.Dependency: 提取到的依赖列表
//   - error: 错误信息
func (e *SconsExtractor) Extract(projectPath string, filePath string) ([]models.Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	var dependencies []models.Dependency
	scanner := bufio.NewScanner(file)

	// 用于匹配各种 SCons 依赖声明的正则表达式
	envRegex := regexp.MustCompile(`env\s*=\s*Environment\s*\((.*?)\)`)
	dependsRegex := regexp.MustCompile(`Depends\s*\(\s*([^,]+)\s*,\s*([^)]+)\s*\)`)
	requiresRegex := regexp.MustCompile(`Requires\s*\(\s*([^,]+)\s*,\s*([^)]+)\s*\)`)
	importRegex := regexp.MustCompile(`Import\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	sconscriptRegex := regexp.MustCompile(`SConscript\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	libraryRegex := regexp.MustCompile(`env\.Library\s*\(\s*['"]([^'"]+)['"]\s*,\s*([^)]+)\s*\)`)
	programRegex := regexp.MustCompile(`env\.Program\s*\(\s*['"]([^'"]+)['"]\s*,\s*([^)]+)\s*\)`)
	parseConfigRegex := regexp.MustCompile(`env\.ParseConfig\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	var currentTarget string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 处理 Environment 配置
		if matches := envRegex.FindStringSubmatch(line); len(matches) > 1 {
			envConfig := matches[1]
			deps := extractEnvironmentDeps(envConfig)
			for _, dep := range deps {
				dependencies = append(dependencies, models.Dependency{
					Name:     dep,
					Type:     "scons_env",
					FilePath: filePath,
					Line:     lineNum,
				})
			}
			continue
		}

		// 处理 Depends 声明
		if matches := dependsRegex.FindStringSubmatch(line); len(matches) > 2 {
			target := strings.TrimSpace(matches[1])
			deps := extractListItems(matches[2])
			for _, dep := range deps {
				dependencies = append(dependencies, models.Dependency{
					Name:     dep,
					Type:     "scons_depends",
					FilePath: filePath,
					Line:     lineNum,
					Parent:   target,
				})
			}
			continue
		}

		// 处理 Requires 声明
		if matches := requiresRegex.FindStringSubmatch(line); len(matches) > 2 {
			target := strings.TrimSpace(matches[1])
			deps := extractListItems(matches[2])
			for _, dep := range deps {
				dependencies = append(dependencies, models.Dependency{
					Name:     dep,
					Type:     "scons_requires",
					FilePath: filePath,
					Line:     lineNum,
					Parent:   target,
				})
			}
			continue
		}

		// 处理 Import 声明
		if matches := importRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "scons_import",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
			continue
		}

		// 处理 SConscript 声明
		if matches := sconscriptRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "scons_script",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
			continue
		}

		// 处理 Library 声明
		if matches := libraryRegex.FindStringSubmatch(line); len(matches) > 2 {
			target := matches[1]
			sources := extractListItems(matches[2])
			for _, source := range sources {
				dep := models.Dependency{
					Name:     source,
					Type:     "scons_library",
					FilePath: filePath,
					Line:     lineNum,
					Parent:   target,
				}
				dependencies = append(dependencies, dep)
			}
			continue
		}

		// 处理 Program 声明
		if matches := programRegex.FindStringSubmatch(line); len(matches) > 2 {
			target := matches[1]
			sources := extractListItems(matches[2])
			for _, source := range sources {
				dep := models.Dependency{
					Name:     source,
					Type:     "scons_program",
					FilePath: filePath,
					Line:     lineNum,
					Parent:   target,
				}
				dependencies = append(dependencies, dep)
			}
			continue
		}

		// 处理 ParseConfig 声明
		if matches := parseConfigRegex.FindStringSubmatch(line); len(matches) > 1 {
			command := matches[1]
			if strings.Contains(command, "pkg-config") {
				pkgs := extractPkgConfigPackages(command)
				for _, pkg := range pkgs {
					dep := models.Dependency{
						Name:     pkg,
						Type:     "scons_pkg_config",
						FilePath: filePath,
						Line:     lineNum,
					}
					dependencies = append(dependencies, dep)
				}
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %v", filePath, err)
	}

	return dependencies, nil
}

// extractEnvironmentDeps 从 Environment 配置中提取依赖
func extractEnvironmentDeps(config string) []string {
	var deps []string
	pkgConfigRegex := regexp.MustCompile(`LIBS\s*=\s*\[([^]]+)\]`)
	if matches := pkgConfigRegex.FindStringSubmatch(config); len(matches) > 1 {
		deps = extractListItems(matches[1])
	}
	return deps
}

// extractListItems 从列表字符串中提取项目
func extractListItems(list string) []string {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil
	}

	// 移除列表的方括号
	list = strings.TrimPrefix(list, "[")
	list = strings.TrimSuffix(list, "]")

	// 分割并清理每个项目
	var items []string
	for _, item := range strings.Split(list, ",") {
		item = strings.Trim(item, " '\"`")
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

// extractPkgConfigPackages 从 pkg-config 命令中提取包名
func extractPkgConfigPackages(command string) []string {
	var pkgs []string
	pkgRegex := regexp.MustCompile(`pkg-config\s+--[^\s]+\s+([^'"\s]+)`)
	if matches := pkgRegex.FindStringSubmatch(command); len(matches) > 1 {
		pkgs = strings.Split(matches[1], " ")
	}
	return pkgs
}

// GetName 返回提取器的名称
func (e *SconsExtractor) GetName() string {
	return e.name
}

// IsApplicable 检查提取器是否适用于指定的文件
func (e *SconsExtractor) IsApplicable(filePath string) bool {
	fileName := filepath.Base(filePath)
	return e.filePattern.MatchString(fileName)
}

// GetPriority 返回提取器的优先级
func (e *SconsExtractor) GetPriority() int {
	return 100
}

// GetFilePattern 返回提取器的文件模式
func (e *SconsExtractor) GetFilePattern() *regexp.Regexp {
	return e.filePattern
}

// String 返回提取器的字符串表示
func (e *SconsExtractor) String() string {
	return fmt.Sprintf("SconsExtractor{name: %s, pattern: %s}", e.name, e.filePattern.String())
}

// 注意事项:
// 1. SCons 构建文件是 Python 脚本,需要处理 Python 语法
// 2. 需要处理多种依赖声明方式
// 3. 需要处理环境变量和配置
// 4. 需要处理子脚本包含
// 5. 建议添加对自定义构建器的支持

// 使用示例:
// extractor := NewSconsExtractor()
// deps, err := extractor.Extract("/path/to/project", "/path/to/project/SConstruct")
// if err != nil {
//     log.Fatal(err)
// }
// for _, dep := range deps {
//     fmt.Printf("Found dependency: %s (%s)\n", dep.Name, dep.Type)
// } 