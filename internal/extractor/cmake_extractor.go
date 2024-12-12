package extractor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// CMakeExtractor CMake依赖提取器
type CMakeExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewCMakeExtractor 创建CMake提取器
func NewCMakeExtractor(path string) *CMakeExtractor {
	return &CMakeExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取CMake依赖
func (e *CMakeExtractor) Extract() ([]models.Dependency, error) {
	// 读取CMake文件
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(CMakeExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	// 正则表达式
	findPackageRe := regexp.MustCompile(`(?i)find_package\s*\(\s*(\w+)`)
	findLibraryRe := regexp.MustCompile(`(?i)find_library\s*\(\s*\w+\s+(\w+)`)
	targetLinkRe := regexp.MustCompile(`(?i)target_link_libraries\s*\(\s*\w+\s+(?:PRIVATE|PUBLIC|INTERFACE)?\s*([^)]+)\)`)
	includeRe := regexp.MustCompile(`(?i)include\s*\(\s*(\w+)`)
	requireRe := regexp.MustCompile(`(?i)require\s*\(\s*(\w+)`)

	var multiLineComment bool
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// 处理多行注释
		if strings.Contains(line, "/*") {
			multiLineComment = true
		}
		if multiLineComment {
			if strings.Contains(line, "*/") {
				multiLineComment = false
			}
			continue
		}

		// 忽略单行注释
		if e.config.IgnoreComments {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
		}

		// 提取find_package
		if matches := findPackageRe.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.NewDependency(matches[1])
			dep.Type = "package"
			dep.BuildSystem = "cmake"
			dep.DetectedBy = "CMakeExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "CMakeLists.txt"
			deps = append(deps, *dep)
		}

		// 提取find_library
		if matches := findLibraryRe.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.NewDependency(matches[1])
			dep.Type = "library"
			dep.BuildSystem = "cmake"
			dep.DetectedBy = "CMakeExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "CMakeLists.txt"
			deps = append(deps, *dep)
		}

		// 提取target_link_libraries
		if matches := targetLinkRe.FindStringSubmatch(line); len(matches) > 1 {
			libs := strings.Fields(matches[1])
			for _, lib := range libs {
				// 忽略变量引用
				if strings.HasPrefix(lib, "${") {
					continue
				}
				dep := models.NewDependency(lib)
				dep.Type = "library"
				dep.BuildSystem = "cmake"
				dep.DetectedBy = "CMakeExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "CMakeLists.txt"
				deps = append(deps, *dep)
			}
		}

		// 提取include
		if matches := includeRe.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.NewDependency(matches[1])
			dep.Type = "module"
			dep.BuildSystem = "cmake"
			dep.DetectedBy = "CMakeExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "CMakeLists.txt"
			deps = append(deps, *dep)
		}

		// 提取require
		if matches := requireRe.FindStringSubmatch(line); len(matches) > 1 {
			dep := models.NewDependency(matches[1])
			dep.Type = "requirement"
			dep.BuildSystem = "cmake"
			dep.DetectedBy = "CMakeExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "CMakeLists.txt"
			deps = append(deps, *dep)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(CMakeExtractorType, e.FilePath, err.Error())
	}

	// 递归处理包含的CMake文件
	if e.config.MaxDepth > 0 {
		e.config.MaxDepth--
		if err := e.extractIncludedFiles(&deps); err != nil {
			return nil, err
		}
	}

	return deps, nil
}

// extractIncludedFiles 提取包含的CMake文件中的依赖
func (e *CMakeExtractor) extractIncludedFiles(deps *[]models.Dependency) error {
	dir := filepath.Dir(e.FilePath)
	includeRe := regexp.MustCompile(`(?i)include\s*\(\s*([^)]+)\)`)

	file, err := os.Open(e.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// 查找include语句
		if matches := includeRe.FindStringSubmatch(line); len(matches) > 1 {
			includePath := strings.Trim(matches[1], `"'`)
			// 忽略内置模块
			if strings.HasPrefix(includePath, "${") {
				continue
			}

			// 构建完整路径
			fullPath := filepath.Join(dir, includePath)
			if !strings.HasSuffix(fullPath, ".cmake") {
				fullPath += ".cmake"
			}

			// 检查文件是否存在
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				continue
			}

			// 创建新的提取器处理包含的文件
			includeExtractor := NewCMakeExtractor(fullPath)
			includeExtractor.config = e.config

			// 提取依赖
			includeDeps, err := includeExtractor.Extract()
			if err != nil {
				return fmt.Errorf("failed to extract dependencies from included file %s: %v", fullPath, err)
			}

			*deps = append(*deps, includeDeps...)
		}
	}

	return scanner.Err()
}

// CMakeExtractorFactory CMake提取器工厂
type CMakeExtractorFactory struct{}

// CreateExtractor 创建CMake提取器
func (f *CMakeExtractorFactory) CreateExtractor(path string) Extractor {
	return NewCMakeExtractor(path)
}

func init() {
	// 注册CMake提取器
	RegisterExtractor(CMakeExtractorType, &CMakeExtractorFactory{})
}

/*
使用示例:

1. 创建CMake提取器:
extractor := NewCMakeExtractor("CMakeLists.txt")

2. 配置提取器:
extractor.config.IgnoreComments = true
extractor.config.MaxDepth = 5

3. 提取依赖:
deps, err := extractor.Extract()
if err != nil {
    log.Printf("Failed to extract dependencies: %v\n", err)
    return
}

4. 处理依赖信息:
for _, dep := range deps {
    fmt.Printf("Found dependency: %s (%s)\n", dep.Name, dep.Type)
}

示例CMakeLists.txt文件:
```cmake
cmake_minimum_required(VERSION 3.10)
project(MyProject)

# 查找包
find_package(Boost REQUIRED)
find_package(OpenCV REQUIRED)

# 查找库
find_library(MATH_LIBRARY m)

# 包含模块
include(CTest)
include(MyCustomModule)

# 链接库
target_link_libraries(MyTarget
    PRIVATE
        Boost::boost
        OpenCV::OpenCV
        ${MATH_LIBRARY}
)
```
*/ 