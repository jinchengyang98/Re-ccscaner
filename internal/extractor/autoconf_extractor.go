package extractor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// AutoconfExtractor Autoconf依赖提取器
type AutoconfExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewAutoconfExtractor 创建Autoconf提取器
func NewAutoconfExtractor(path string) *AutoconfExtractor {
	return &AutoconfExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取Autoconf依赖
func (e *AutoconfExtractor) Extract() ([]models.Dependency, error) {
	// 读取configure.ac或configure.in文件
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(AutoconfExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	// 正则表达式
	pkgCheckRe := regexp.MustCompile(`PKG_CHECK_MODULES\s*\(\s*\w+\s*,\s*\[([^\]]+)\]`)
	acCheckLibRe := regexp.MustCompile(`AC_CHECK_LIB\s*\(\s*([^,\s]+)`)
	acCheckHeaderRe := regexp.MustCompile(`AC_CHECK_HEADER\s*\(\s*([^,\s]+)`)
	acPathProgRe := regexp.MustCompile(`AC_PATH_PROG\s*\(\s*\w+\s*,\s*([^,\s]+)`)
	amInitRe := regexp.MustCompile(`AM_INIT_AUTOMAKE\s*\(\s*([^,\s]+)\s*,\s*([^,\s\)]+)`)
	acInitRe := regexp.MustCompile(`AC_INIT\s*\(\s*([^,\s]+)\s*,\s*([^,\s\)]+)`)
	acConfigRe := regexp.MustCompile(`AC_CONFIG_SUBDIRS\s*\(\s*([^,\s\)]+)`)

	var multiLineComment bool
	var continuationLine string
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// 处理多行注释
		if strings.Contains(line, "dnl") {
			continue
		}

		// 处理行继续符
		if strings.HasSuffix(line, "\\") {
			continuationLine += strings.TrimSuffix(line, "\\") + " "
			continue
		} else if continuationLine != "" {
			line = continuationLine + line
			continuationLine = ""
		}

		// 提取PKG_CHECK_MODULES
		if matches := pkgCheckRe.FindStringSubmatch(line); len(matches) > 1 {
			pkgs := strings.Split(matches[1], " ")
			for _, pkg := range pkgs {
				pkg = strings.TrimSpace(pkg)
				if pkg == "" {
					continue
				}

				// 解析包名和版本要求
				parts := strings.Split(pkg, ">=")
				name := parts[0]
				version := ""
				if len(parts) > 1 {
					version = parts[1]
				}

				dep := models.NewDependency(name)
				dep.Version = version
				dep.Type = "package"
				dep.BuildSystem = "autoconf"
				dep.DetectedBy = "AutoconfExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "configure.ac"
				deps = append(deps, *dep)
			}
		}

		// 提取AC_CHECK_LIB
		if matches := acCheckLibRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.Trim(matches[1], `"'`)
			dep := models.NewDependency(name)
			dep.Type = "library"
			dep.BuildSystem = "autoconf"
			dep.DetectedBy = "AutoconfExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "configure.ac"
			deps = append(deps, *dep)
		}

		// 提取AC_CHECK_HEADER
		if matches := acCheckHeaderRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.Trim(matches[1], `"'`)
			dep := models.NewDependency(name)
			dep.Type = "header"
			dep.BuildSystem = "autoconf"
			dep.DetectedBy = "AutoconfExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "configure.ac"
			deps = append(deps, *dep)
		}

		// 提取AC_PATH_PROG
		if matches := acPathProgRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.Trim(matches[1], `"'`)
			dep := models.NewDependency(name)
			dep.Type = "program"
			dep.BuildSystem = "autoconf"
			dep.DetectedBy = "AutoconfExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "configure.ac"
			deps = append(deps, *dep)
		}

		// 提取AM_INIT_AUTOMAKE
		if matches := amInitRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.Trim(matches[1], `"'`)
			version := ""
			if len(matches) > 2 {
				version = strings.Trim(matches[2], `"'`)
			}
			dep := models.NewDependency("automake")
			dep.Version = version
			dep.Type = "build_system"
			dep.BuildSystem = "autoconf"
			dep.DetectedBy = "AutoconfExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "configure.ac"
			deps = append(deps, *dep)
		}

		// 提取AC_INIT
		if matches := acInitRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.Trim(matches[1], `"'`)
			version := ""
			if len(matches) > 2 {
				version = strings.Trim(matches[2], `"'`)
			}
			dep := models.NewDependency("autoconf")
			dep.Version = version
			dep.Type = "build_system"
			dep.BuildSystem = "autoconf"
			dep.DetectedBy = "AutoconfExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "configure.ac"
			deps = append(deps, *dep)
		}

		// 提取AC_CONFIG_SUBDIRS
		if matches := acConfigRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.Trim(matches[1], `"'`)
			dep := models.NewDependency(name)
			dep.Type = "subproject"
			dep.BuildSystem = "autoconf"
			dep.DetectedBy = "AutoconfExtractor"
			dep.ConfigFile = e.FilePath
			dep.ConfigFileType = "configure.ac"
			deps = append(deps, *dep)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(AutoconfExtractorType, e.FilePath, err.Error())
	}

	return deps, nil
}

// AutoconfExtractorFactory Autoconf提取器工厂
type AutoconfExtractorFactory struct{}

// CreateExtractor 创建Autoconf提取器
func (f *AutoconfExtractorFactory) CreateExtractor(path string) Extractor {
	return NewAutoconfExtractor(path)
}

func init() {
	// 注册Autoconf提取器
	RegisterExtractor(AutoconfExtractorType, &AutoconfExtractorFactory{})
}

/*
使用示例:

1. 创建Autoconf提取器:
extractor := NewAutoconfExtractor("configure.ac")

2. 提取依赖:
deps, err := extractor.Extract()
if err != nil {
    log.Printf("Failed to extract dependencies: %v\n", err)
    return
}

3. 处理依赖信息:
for _, dep := range deps {
    fmt.Printf("Found dependency: %s %s (%s)\n", dep.Name, dep.Version, dep.Type)
}

示例configure.ac文件:
```autoconf
AC_INIT([myproject], [1.0.0])
AM_INIT_AUTOMAKE([myproject], [1.0.0])

# 检查编译器
AC_PROG_CC
AC_PROG_CXX

# 检查程序
AC_PATH_PROG([PYTHON], [python3])
AC_PATH_PROG([DOXYGEN], [doxygen])

# 检查库
AC_CHECK_LIB([m], [cos])
AC_CHECK_LIB([pthread], [pthread_create])

# 检查头文件
AC_CHECK_HEADER([stdio.h])
AC_CHECK_HEADER([stdlib.h])

# 检查pkg-config模块
PKG_CHECK_MODULES([GTK], [gtk+-3.0 >= 3.20])
PKG_CHECK_MODULES([GLIB], [glib-2.0 >= 2.50])

# 配置子目录
AC_CONFIG_SUBDIRS([lib/mylib])
AC_CONFIG_SUBDIRS([tests])

AC_OUTPUT
```
*/ 