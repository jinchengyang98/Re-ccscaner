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

// NinjaExtractor 实现了从 Ninja 构建文件中提取依赖信息的提取器
type NinjaExtractor struct {
	BaseExtractor
}

// NewNinjaExtractor 创建一个新的 Ninja 提取器实例
func NewNinjaExtractor() *NinjaExtractor {
	return &NinjaExtractor{
		BaseExtractor: BaseExtractor{
			name:        "Ninja",
			filePattern: regexp.MustCompile(`build\.ninja|\.ninja$`),
		},
	}
}

// Extract 从 Ninja 构建文件中提取依赖信息
// 参数:
//   - projectPath: 项目根目录路径
//   - filePath: Ninja 构建文件路径
//
// 返回:
//   - []models.Dependency: 提取到的依赖列表
//   - error: 错误信息
func (e *NinjaExtractor) Extract(projectPath string, filePath string) ([]models.Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	var dependencies []models.Dependency
	scanner := bufio.NewScanner(file)

	// 用于匹配各种 Ninja 规则和依赖的正则表达式
	buildRuleRegex := regexp.MustCompile(`^build\s+([^:]+):\s+([^\s]+)\s+([^|#]+)(?:\|\s+([^#]+))?(?:#.*)?$`)
	includeRegex := regexp.MustCompile(`^include\s+([^#]+)(?:#.*)?$`)
	subninjaDirRegex := regexp.MustCompile(`^subninja\s+([^#]+)(?:#.*)?$`)
	variableRegex := regexp.MustCompile(`^\s*([^=]+)\s*=\s*([^#]+)(?:#.*)?$`)

	var currentRule string
	var variables = make(map[string]string)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 处理变量定义
		if matches := variableRegex.FindStringSubmatch(line); len(matches) > 2 {
			name := strings.TrimSpace(matches[1])
			value := strings.TrimSpace(matches[2])
			variables[name] = value
			continue
		}

		// 处理 include 指令
		if matches := includeRegex.FindStringSubmatch(line); len(matches) > 1 {
			includePath := strings.TrimSpace(matches[1])
			includePath = expandVariables(includePath, variables)
			dep := models.Dependency{
				Name:     includePath,
				Type:     "ninja_include",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
			continue
		}

		// 处理 subninja 指令
		if matches := subninjaDirRegex.FindStringSubmatch(line); len(matches) > 1 {
			subninjaPath := strings.TrimSpace(matches[1])
			subninjaPath = expandVariables(subninjaPath, variables)
			dep := models.Dependency{
				Name:     subninjaPath,
				Type:     "ninja_subninja",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
			continue
		}

		// 处理构建规则
		if matches := buildRuleRegex.FindStringSubmatch(line); len(matches) > 3 {
			outputs := strings.Fields(matches[1])
			rule := matches[2]
			inputs := strings.Fields(matches[3])
			var implicitDeps []string
			if len(matches) > 4 && matches[4] != "" {
				implicitDeps = strings.Fields(matches[4])
			}

			currentRule = rule

			// 添加输入依赖
			for _, input := range inputs {
				input = expandVariables(input, variables)
				dep := models.Dependency{
					Name:     input,
					Type:     "ninja_input",
					FilePath: filePath,
					Line:     lineNum,
					Parent:   outputs[0], // 使用第一个输出作为父节点
					Rule:     rule,
				}
				dependencies = append(dependencies, dep)
			}

			// 添加隐式依赖
			for _, implicit := range implicitDeps {
				implicit = expandVariables(implicit, variables)
				dep := models.Dependency{
					Name:     implicit,
					Type:     "ninja_implicit",
					FilePath: filePath,
					Line:     lineNum,
					Parent:   outputs[0],
					Rule:     rule,
				}
				dependencies = append(dependencies, dep)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %v", filePath, err)
	}

	return dependencies, nil
}

// expandVariables 展开变量引用
func expandVariables(value string, variables map[string]string) string {
	for name, val := range variables {
		value = strings.ReplaceAll(value, "$"+name, val)
		value = strings.ReplaceAll(value, "${"+name+"}", val)
	}
	return value
}

// GetName 返回提取器的名称
func (e *NinjaExtractor) GetName() string {
	return e.name
}

// IsApplicable 检查提取器是否适用于指定的文件
func (e *NinjaExtractor) IsApplicable(filePath string) bool {
	fileName := filepath.Base(filePath)
	return e.filePattern.MatchString(fileName)
}

// GetPriority 返回提取器的优先级
func (e *NinjaExtractor) GetPriority() int {
	return 100
}

// GetFilePattern 返回提取器的文件模式
func (e *NinjaExtractor) GetFilePattern() *regexp.Regexp {
	return e.filePattern
}

// String 返回提取器的字符串表示
func (e *NinjaExtractor) String() string {
	return fmt.Sprintf("NinjaExtractor{name: %s, pattern: %s}", e.name, e.filePattern.String())
}

// 注意事项:
// 1. Ninja 构建文件通常由其他构建系统生成
// 2. 需要处理变量展开
// 3. 需要处理 include 和 subninja 指令
// 4. 需要处理显式和隐式依赖
// 5. 建议添加对构建规则的分析

// 使用示例:
// extractor := NewNinjaExtractor()
// deps, err := extractor.Extract("/path/to/project", "/path/to/project/build.ninja")
// if err != nil {
//     log.Fatal(err)
// }
// for _, dep := range deps {
//     fmt.Printf("Found dependency: %s (%s)\n", dep.Name, dep.Type)
// } 