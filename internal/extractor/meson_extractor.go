package extractor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// MesonExtractor Meson依赖提取器
type MesonExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewMesonExtractor 创建Meson提取器
func NewMesonExtractor(path string) *MesonExtractor {
	return &MesonExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取Meson依赖
func (e *MesonExtractor) Extract() ([]models.Dependency, error) {
	// 读取meson.build文件
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(MesonExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	// 正则表达式
	dependencyRe := regexp.MustCompile(`dependency\s*\(\s*['"]([^'"]+)['"]\s*(?:,\s*version\s*:\s*['"]([^'"]+)['"])?\s*\)`)
	pkgConfigRe := regexp.MustCompile(`pkg\.get_variable\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	requireRe := regexp.MustCompile(`requires\s*:\s*\[\s*(['"][^'"]+['"](?:\s*,\s*['"][^'"]+['"])*)\s*\]`)
	subprojectRe := regexp.MustCompile(`subproject\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	var multiLineComment bool
	var continuationLine string
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

		// 处理行继续符
		if strings.HasSuffix(line, "\\") {
			continuationLine += strings.TrimSuffix(line, "\\") + " "
			continue
		} else if continuationLine != "" {
			line = continuationLine + line
			continuationLine = ""
		}

		// 提取dependency()调用
		if matches := dependencyRe.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				dep := models.NewDependency(match[1])
				if len(match) > 2 && match[2] != "" {
					dep.Version = match[2]
				}
				dep.Type = "dependency"
				dep.BuildSystem = "meson"
				dep.DetectedBy = "MesonExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "meson.build"
				deps = append(deps, *dep)
			}
		}

		// 提取pkg-config变量
		if matches := pkgConfigRe.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				dep := models.NewDependency(match[1])
				dep.Type = "pkgconfig"
				dep.BuildSystem = "meson"
				dep.DetectedBy = "MesonExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "meson.build"
				deps = append(deps, *dep)
			}
		}

		// 提取requires列表
		if matches := requireRe.FindStringSubmatch(line); len(matches) > 1 {
			requirements := strings.Split(matches[1], ",")
			for _, req := range requirements {
				req = strings.Trim(strings.TrimSpace(req), "'\"")
				dep := models.NewDependency(req)
				dep.Type = "requirement"
				dep.BuildSystem = "meson"
				dep.DetectedBy = "MesonExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "meson.build"
				deps = append(deps, *dep)
			}
		}

		// 提取子项目
		if matches := subprojectRe.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				dep := models.NewDependency(match[1])
				dep.Type = "subproject"
				dep.BuildSystem = "meson"
				dep.DetectedBy = "MesonExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "meson.build"
				deps = append(deps, *dep)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(MesonExtractorType, e.FilePath, err.Error())
	}

	return deps, nil
}

// MesonExtractorFactory Meson提取器工厂
type MesonExtractorFactory struct{}

// CreateExtractor 创建Meson提取器
func (f *MesonExtractorFactory) CreateExtractor(path string) Extractor {
	return NewMesonExtractor(path)
}

func init() {
	// 注册Meson提取器
	RegisterExtractor(MesonExtractorType, &MesonExtractorFactory{})
}

/*
使用示例:

1. 创建Meson提取器:
extractor := NewMesonExtractor("meson.build")

2. 配置提取器:
extractor.config.IgnoreComments = true

3. 提取依赖:
deps, err := extractor.Extract()
if err != nil {
    log.Printf("Failed to extract dependencies: %v\n", err)
    return
}

4. 处理依赖信息:
for _, dep := range deps {
    fmt.Printf("Found dependency: %s %s (%s)\n", dep.Name, dep.Version, dep.Type)
}

示例meson.build文件:
```meson
project('myproject', 'cpp',
  version : '0.1',
  default_options : ['cpp_std=c++17']
)

# 基本依赖
boost_dep = dependency('boost', version : '>=1.74')
openssl_dep = dependency('openssl', version : '>=1.1')
threads_dep = dependency('threads')

# pkg-config依赖
gtk_dep = dependency('gtk+-3.0')
pkg = import('pkgconfig')
gtk_version = pkg.get_variable('gtk+-3.0', 'Version')

# 子项目
json_proj = subproject('json')
json_dep = json_proj.get_variable('json_dep')

# 依赖列表
project_deps = [
    boost_dep,
    openssl_dep,
    threads_dep,
    gtk_dep,
    json_dep,
]

executable('myapp',
    sources,
    dependencies : project_deps,
    install : true
)
```
*/ 