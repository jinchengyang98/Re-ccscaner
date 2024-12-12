package extractor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// PkgConfigExtractor PkgConfig依赖提取器
type PkgConfigExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewPkgConfigExtractor 创建PkgConfig提取器
func NewPkgConfigExtractor(path string) *PkgConfigExtractor {
	return &PkgConfigExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取PkgConfig依赖
func (e *PkgConfigExtractor) Extract() ([]models.Dependency, error) {
	// 读取.pc文件
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(PkgConfigExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	// 正则表达式
	nameRe := regexp.MustCompile(`^Name:\s*(.+)`)
	versionRe := regexp.MustCompile(`^Version:\s*(.+)`)
	descriptionRe := regexp.MustCompile(`^Description:\s*(.+)`)
	urlRe := regexp.MustCompile(`^URL:\s*(.+)`)
	requiresRe := regexp.MustCompile(`^Requires(?:\.private)?:\s*(.+)`)
	conflictsRe := regexp.MustCompile(`^Conflicts:\s*(.+)`)
	libsRe := regexp.MustCompile(`^Libs(?:\.private)?:\s*(.+)`)
	cflagsRe := regexp.MustCompile(`^Cflags:\s*(.+)`)

	var currentDep *models.Dependency
	var variables = make(map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 忽略空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 处理变量定义
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				variables[key] = value
				continue
			}
		}

		// 替换变量引用
		for key, value := range variables {
			line = strings.ReplaceAll(line, "${"+key+"}", value)
			line = strings.ReplaceAll(line, "$"+key, value)
		}

		// 提取包名
		if matches := nameRe.FindStringSubmatch(line); len(matches) > 1 {
			if currentDep == nil {
				currentDep = models.NewDependency(matches[1])
				currentDep.Type = "package"
				currentDep.BuildSystem = "pkgconfig"
				currentDep.DetectedBy = "PkgConfigExtractor"
				currentDep.ConfigFile = e.FilePath
				currentDep.ConfigFileType = ".pc"
			}
			continue
		}

		if currentDep == nil {
			continue
		}

		// 提取版本
		if matches := versionRe.FindStringSubmatch(line); len(matches) > 1 {
			currentDep.Version = matches[1]
			continue
		}

		// 提取描述
		if matches := descriptionRe.FindStringSubmatch(line); len(matches) > 1 {
			currentDep.Description = matches[1]
			continue
		}

		// 提取URL
		if matches := urlRe.FindStringSubmatch(line); len(matches) > 1 {
			currentDep.Homepage = matches[1]
			continue
		}

		// 提取依赖
		if matches := requiresRe.FindStringSubmatch(line); len(matches) > 1 {
			reqs := strings.Split(matches[1], ",")
			for _, req := range reqs {
				req = strings.TrimSpace(req)
				if req == "" {
					continue
				}

				// 解析版本约束
				parts := strings.Fields(req)
				name := parts[0]
				var constraints []models.VersionConstrain

				for i := 1; i < len(parts); i++ {
					op := parts[i]
					if i+1 < len(parts) && (op == ">" || op == ">=" || op == "=" || op == "<" || op == "<=") {
						constraints = append(constraints, models.VersionConstrain{
							Operator: op,
							Version:  parts[i+1],
						})
						i++
					}
				}

				dep := models.NewDependency(name)
				dep.Type = "requirement"
				dep.BuildSystem = "pkgconfig"
				dep.DetectedBy = "PkgConfigExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = ".pc"
				dep.Constraints = constraints

				deps = append(deps, *dep)
			}
			continue
		}

		// 提取冲突
		if matches := conflictsRe.FindStringSubmatch(line); len(matches) > 1 {
			conflicts := strings.Split(matches[1], ",")
			for _, conflict := range conflicts {
				currentDep.Conflicts = append(currentDep.Conflicts, strings.TrimSpace(conflict))
			}
			continue
		}

		// 提取库标志
		if matches := libsRe.FindStringSubmatch(line); len(matches) > 1 {
			currentDep.BuildFlags = append(currentDep.BuildFlags, strings.Fields(matches[1])...)
			continue
		}

		// 提取编译标志
		if matches := cflagsRe.FindStringSubmatch(line); len(matches) > 1 {
			currentDep.BuildFlags = append(currentDep.BuildFlags, strings.Fields(matches[1])...)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(PkgConfigExtractorType, e.FilePath, err.Error())
	}

	// 添加主包依赖
	if currentDep != nil {
		deps = append(deps, *currentDep)
	}

	return deps, nil
}

// PkgConfigExtractorFactory PkgConfig提取器工厂
type PkgConfigExtractorFactory struct{}

// CreateExtractor 创建PkgConfig提取器
func (f *PkgConfigExtractorFactory) CreateExtractor(path string) Extractor {
	return NewPkgConfigExtractor(path)
}

func init() {
	// 注册PkgConfig提取器
	RegisterExtractor(PkgConfigExtractorType, &PkgConfigExtractorFactory{})
}

/*
使用示例:

1. 创建PkgConfig提取器:
extractor := NewPkgConfigExtractor("libfoo.pc")

2. 提取依赖:
deps, err := extractor.Extract()
if err != nil {
    log.Printf("Failed to extract dependencies: %v\n", err)
    return
}

3. 处理依赖信息:
for _, dep := range deps {
    fmt.Printf("Found package: %s %s\n", dep.Name, dep.Version)
    if len(dep.Constraints) > 0 {
        fmt.Printf("  Version constraints:\n")
        for _, c := range dep.Constraints {
            fmt.Printf("    %s %s\n", c.Operator, c.Version)
        }
    }
    if len(dep.BuildFlags) > 0 {
        fmt.Printf("  Build flags: %v\n", dep.BuildFlags)
    }
}

示例.pc文件:
```
prefix=/usr/local
exec_prefix=${prefix}
libdir=${exec_prefix}/lib
includedir=${prefix}/include

Name: libfoo
Description: A library for doing foo things
Version: 1.2.3
URL: https://example.com/foo

Requires: libbar >= 2.0.0, libqux
Requires.private: libinternal >= 1.0.0
Conflicts: libold < 3.0.0

Libs: -L${libdir} -lfoo
Libs.private: -lm
Cflags: -I${includedir}/foo -DFOO_ENABLE
```
*/ 