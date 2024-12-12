package extractor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// MakeExtractor Make依赖提取器
type MakeExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewMakeExtractor 创建Make提取器
func NewMakeExtractor(path string) *MakeExtractor {
	return &MakeExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取Make依赖
func (e *MakeExtractor) Extract() ([]models.Dependency, error) {
	// 读取Makefile
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(MakeExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	// 正则表达式
	libRe := regexp.MustCompile(`-l(\w+)`)
	pkgConfigRe := regexp.MustCompile(`pkg-config\s+--libs\s+(\S+)`)
	includeRe := regexp.MustCompile(`-I(\S+)`)
	requireRe := regexp.MustCompile(`REQUIRES\s*=\s*(.+)`)
	dependsRe := regexp.MustCompile(`DEPENDS\s*=\s*(.+)`)

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

		// 提取库依赖(-l选项)
		if matches := libRe.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				dep := models.NewDependency(match[1])
				dep.Type = "library"
				dep.BuildSystem = "make"
				dep.DetectedBy = "MakeExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "Makefile"
				deps = append(deps, *dep)
			}
		}

		// 提取pkg-config依赖
		if matches := pkgConfigRe.FindStringSubmatch(line); len(matches) > 1 {
			pkgs := strings.Fields(matches[1])
			for _, pkg := range pkgs {
				dep := models.NewDependency(pkg)
				dep.Type = "package"
				dep.BuildSystem = "make"
				dep.DetectedBy = "MakeExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "Makefile"
				deps = append(deps, *dep)
			}
		}

		// 提取包含路径依赖(-I选项)
		if matches := includeRe.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				path := match[1]
				// 忽略系统路径
				if strings.HasPrefix(path, "/usr/include") {
					continue
				}
				dep := models.NewDependency(path)
				dep.Type = "include"
				dep.BuildSystem = "make"
				dep.DetectedBy = "MakeExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "Makefile"
				deps = append(deps, *dep)
			}
		}

		// 提取REQUIRES依赖
		if matches := requireRe.FindStringSubmatch(line); len(matches) > 1 {
			reqs := strings.Fields(matches[1])
			for _, req := range reqs {
				dep := models.NewDependency(req)
				dep.Type = "requirement"
				dep.BuildSystem = "make"
				dep.DetectedBy = "MakeExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "Makefile"
				deps = append(deps, *dep)
			}
		}

		// 提取DEPENDS依赖
		if matches := dependsRe.FindStringSubmatch(line); len(matches) > 1 {
			depends := strings.Fields(matches[1])
			for _, depend := range depends {
				dep := models.NewDependency(depend)
				dep.Type = "dependency"
				dep.BuildSystem = "make"
				dep.DetectedBy = "MakeExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "Makefile"
				deps = append(deps, *dep)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(MakeExtractorType, e.FilePath, err.Error())
	}

	return deps, nil
}

// MakeExtractorFactory Make提取器工厂
type MakeExtractorFactory struct{}

// CreateExtractor 创建Make提取器
func (f *MakeExtractorFactory) CreateExtractor(path string) Extractor {
	return NewMakeExtractor(path)
}

func init() {
	// 注册Make提取器
	RegisterExtractor(MakeExtractorType, &MakeExtractorFactory{})
}

/*
使用示例:

1. 创建Make提取器:
extractor := NewMakeExtractor("Makefile")

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
    fmt.Printf("Found dependency: %s (%s)\n", dep.Name, dep.Type)
}

示例Makefile文件:
```makefile
CC = gcc
CFLAGS = -Wall -O2
LDFLAGS = -lm -lpthread
PKG_CONFIG = pkg-config --libs gtk+-3.0 cairo

INCLUDES = -I/usr/local/include \
          -I../include \
          -I$(HOME)/mylibs/include

REQUIRES = openssl >= 1.0.0 \
          zlib

DEPENDS = libxml2 \
         libcurl

target: $(OBJS)
    $(CC) $(OBJS) $(LDFLAGS) $(shell $(PKG_CONFIG)) -o $@
```
*/ 