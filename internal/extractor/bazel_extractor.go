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

// BazelExtractor 实现了从 Bazel 构建文件中提取依赖信息的提取器
type BazelExtractor struct {
	BaseExtractor
}

// NewBazelExtractor 创建一个新的 Bazel 提取器实例
func NewBazelExtractor() *BazelExtractor {
	return &BazelExtractor{
		BaseExtractor: BaseExtractor{
			name:        "Bazel",
			filePattern: regexp.MustCompile(`BUILD|BUILD\.bazel|WORKSPACE|WORKSPACE\.bazel`),
		},
	}
}

// Extract 从 Bazel 构建文件中提取依赖信息
// 参数:
//   - projectPath: 项目根目录路径
//   - filePath: Bazel 构建文件路径
//
// 返回:
//   - []models.Dependency: 提取到的依赖列表
//   - error: 错误信息
func (e *BazelExtractor) Extract(projectPath string, filePath string) ([]models.Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	var dependencies []models.Dependency
	scanner := bufio.NewScanner(file)

	// 用于匹配各种 Bazel 依赖规则的正则表达式
	httpArchiveRegex := regexp.MustCompile(`http_archive\(\s*name\s*=\s*"([^"]+)"`)
	gitRepositoryRegex := regexp.MustCompile(`git_repository\(\s*name\s*=\s*"([^"]+)"`)
	localRepositoryRegex := regexp.MustCompile(`local_repository\(\s*name\s*=\s*"([^"]+)"`)
	mavenJarRegex := regexp.MustCompile(`maven_jar\(\s*name\s*=\s*"([^"]+)"`)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// 提取 http_archive 依赖
		if matches := httpArchiveRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "bazel_http_archive",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
		}

		// 提取 git_repository 依赖
		if matches := gitRepositoryRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "bazel_git_repository",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
		}

		// 提取 local_repository 依赖
		if matches := localRepositoryRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "bazel_local_repository",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
		}

		// 提取 maven_jar 依赖
		if matches := mavenJarRegex.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.Dependency{
				Name:     matches[1],
				Type:     "bazel_maven_jar",
				FilePath: filePath,
				Line:     lineNum,
			}
			dependencies = append(dependencies, dep)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %v", filePath, err)
	}

	return dependencies, nil
}

// GetName 返回提取器的名称
func (e *BazelExtractor) GetName() string {
	return e.name
}

// IsApplicable 检查提取器是否适用于指定的文件
func (e *BazelExtractor) IsApplicable(filePath string) bool {
	fileName := filepath.Base(filePath)
	return e.filePattern.MatchString(fileName)
}

// GetPriority 返回提取器的优先级
func (e *BazelExtractor) GetPriority() int {
	return 100
}

// GetFilePattern 返回提取器的文件模式
func (e *BazelExtractor) GetFilePattern() *regexp.Regexp {
	return e.filePattern
}

// String 返回提取器的字符串表示
func (e *BazelExtractor) String() string {
	return fmt.Sprintf("BazelExtractor{name: %s, pattern: %s}", e.name, e.filePattern.String())
}

// 注意事项:
// 1. Bazel 构建文件可能包含多种依赖声明方式,这里只实现了最常见的几种
// 2. 实际的 Bazel 构建文件可能更复杂,可能需要处理多行声明、注释等
// 3. 可以根据需要添加更多的依赖类型支持
// 4. 建议添加更多的错误处理和日志记录

// 使用示例:
// extractor := NewBazelExtractor()
// deps, err := extractor.Extract("/path/to/project", "/path/to/project/WORKSPACE")
// if err != nil {
//     log.Fatal(err)
// }
// for _, dep := range deps {
//     fmt.Printf("Found dependency: %s (%s)\n", dep.Name, dep.Type)
// } 