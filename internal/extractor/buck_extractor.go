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

// BuckExtractor 实现了从 Buck 构建文件中提取依赖信息的提取器
type BuckExtractor struct {
	BaseExtractor
}

// NewBuckExtractor 创建一个新的 Buck 提取器实例
func NewBuckExtractor() *BuckExtractor {
	return &BuckExtractor{
		BaseExtractor: BaseExtractor{
			name:        "Buck",
			filePattern: regexp.MustCompile(`BUCK|BUCK\.build|TARGETS|.+\.buck`),
		},
	}
}

// Extract 从 Buck 构建文件中提取依赖信息
// 参数:
//   - projectPath: 项目根目录路径
//   - filePath: Buck 构建文件路径
//
// 返回:
//   - []models.Dependency: 提取到的依赖列表
//   - error: 错误信息
func (e *BuckExtractor) Extract(projectPath string, filePath string) ([]models.Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	var dependencies []models.Dependency
	scanner := bufio.NewScanner(file)

	// 用于匹配各种 Buck 依赖规则的正则表达式
	cppLibraryRegex := regexp.MustCompile(`cpp_library\(\s*name\s*=\s*"([^"]+)"`)
	cppBinaryRegex := regexp.MustCompile(`cpp_binary\(\s*name\s*=\s*"([^"]+)"`)
	prebuiltJarRegex := regexp.MustCompile(`prebuilt_jar\(\s*name\s*=\s*"([^"]+)"`)
	remoteFileRegex := regexp.MustCompile(`remote_file\(\s*name\s*=\s*"([^"]+)"`)
	depsRegex := regexp.MustCompile(`deps\s*=\s*\[(.*?)\]`)

	var currentTarget string
	var inDepsBlock bool
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// 检查是否是新的目标定义
		if matches := cppLibraryRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentTarget = matches[1]
			inDepsBlock = false
		} else if matches := cppBinaryRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentTarget = matches[1]
			inDepsBlock = false
		} else if matches := prebuiltJarRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentTarget = matches[1]
			inDepsBlock = false
		} else if matches := remoteFileRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentTarget = matches[1]
			inDepsBlock = false
		}

		// 检查是否进入依赖块
		if depsRegex.MatchString(line) {
			inDepsBlock = true
			// 提取同一行中的依赖
			if deps := extractDepsFromLine(line); len(deps) > 0 {
				for _, dep := range deps {
					dependencies = append(dependencies, models.Dependency{
						Name:     dep,
						Type:     "buck_dependency",
						FilePath: filePath,
						Line:     lineNum,
						Parent:   currentTarget,
					})
				}
			}
			continue
		}

		// 如果在依赖块中,提取依赖
		if inDepsBlock && line != "]" && line != "" {
			if deps := extractDepsFromLine(line); len(deps) > 0 {
				for _, dep := range deps {
					dependencies = append(dependencies, models.Dependency{
						Name:     dep,
						Type:     "buck_dependency",
						FilePath: filePath,
						Line:     lineNum,
						Parent:   currentTarget,
					})
				}
			}
		}

		// 检查依赖块是否结束
		if inDepsBlock && line == "]" {
			inDepsBlock = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %v", filePath, err)
	}

	return dependencies, nil
}

// extractDepsFromLine 从一行文本中提取依赖
func extractDepsFromLine(line string) []string {
	var deps []string
	// 移除注释
	if idx := strings.Index(line, "//"); idx != -1 {
		line = line[:idx]
	}

	// 提取引号中的依赖名称
	depRegex := regexp.MustCompile(`"([^"]+)"`)
	matches := depRegex.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) > 1 {
			dep := strings.TrimSpace(match[1])
			if dep != "" {
				deps = append(deps, dep)
			}
		}
	}

	return deps
}

// GetName 返回提取器的名称
func (e *BuckExtractor) GetName() string {
	return e.name
}

// IsApplicable 检查提取器是否适用于指定的文件
func (e *BuckExtractor) IsApplicable(filePath string) bool {
	fileName := filepath.Base(filePath)
	return e.filePattern.MatchString(fileName)
}

// GetPriority 返回提取器的优先级
func (e *BuckExtractor) GetPriority() int {
	return 100
}

// GetFilePattern 返回提取器的文件模式
func (e *BuckExtractor) GetFilePattern() *regexp.Regexp {
	return e.filePattern
}

// String 返回提取器的字符串表示
func (e *BuckExtractor) String() string {
	return fmt.Sprintf("BuckExtractor{name: %s, pattern: %s}", e.name, e.filePattern.String())
}

// 注意事项:
// 1. Buck 构建文件使用 Python 语法,可能需要更复杂的解析器来处理所有情况
// 2. 这个实现主要关注最常见的依赖声明方式
// 3. 可能需要处理多行字符串、转义字符等特殊情况
// 4. 建议添加对其他 Buck 规则类型的支持

// 使用示例:
// extractor := NewBuckExtractor()
// deps, err := extractor.Extract("/path/to/project", "/path/to/project/BUCK")
// if err != nil {
//     log.Fatal(err)
// }
// for _, dep := range deps {
//     fmt.Printf("Found dependency: %s (parent: %s)\n", dep.Name, dep.Parent)
// } 