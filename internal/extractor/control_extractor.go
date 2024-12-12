package extractor

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// ControlExtractor Control依赖提取器
type ControlExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewControlExtractor 创建Control提取器
func NewControlExtractor(path string) *ControlExtractor {
	return &ControlExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取Control依赖
func (e *ControlExtractor) Extract() ([]models.Dependency, error) {
	// 读取control文件
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(ControlExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	// 正则表达式
	dependsRe := regexp.MustCompile(`^(?:Build-)?Depends(?:-Indep)?:\s*(.+)`)
	preDepRe := regexp.MustCompile(`^Pre-Depends:\s*(.+)`)
	recommendsRe := regexp.MustCompile(`^Recommends:\s*(.+)`)
	suggestsRe := regexp.MustCompile(`^Suggests:\s*(.+)`)
	enhancesRe := regexp.MustCompile(`^Enhances:\s*(.+)`)
	breaksRe := regexp.MustCompile(`^Breaks:\s*(.+)`)
	conflictsRe := regexp.MustCompile(`^Conflicts:\s*(.+)`)
	providesRe := regexp.MustCompile(`^Provides:\s*(.+)`)
	replacesRe := regexp.MustCompile(`^Replaces:\s*(.+)`)

	var currentPackage string
	var continuationLine string

	for scanner.Scan() {
		line := scanner.Text()

		// 处理行继续符
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continuationLine += " " + strings.TrimSpace(line)
			continue
		} else {
			if continuationLine != "" {
				// 处理前一个继续行
				e.processDependencyLine(continuationLine, currentPackage, &deps)
				continuationLine = ""
			}
		}

		// 检查包名
		if strings.HasPrefix(line, "Package:") {
			currentPackage = strings.TrimSpace(strings.TrimPrefix(line, "Package:"))
			continue
		}

		// 提取依赖关系
		if matches := dependsRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processDependencyLine(matches[1], currentPackage, &deps)
			continue
		}

		// 提取预依赖
		if matches := preDepRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processDependencyLine(matches[1], currentPackage, &deps)
			continue
		}

		// 提取推荐
		if matches := recommendsRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processOptionalDependencyLine(matches[1], currentPackage, "recommends", &deps)
			continue
		}

		// 提取建议
		if matches := suggestsRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processOptionalDependencyLine(matches[1], currentPackage, "suggests", &deps)
			continue
		}

		// 提取增强
		if matches := enhancesRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processOptionalDependencyLine(matches[1], currentPackage, "enhances", &deps)
			continue
		}

		// 提取破坏
		if matches := breaksRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processConflictLine(matches[1], currentPackage, "breaks", &deps)
			continue
		}

		// 提取冲突
		if matches := conflictsRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processConflictLine(matches[1], currentPackage, "conflicts", &deps)
			continue
		}

		// 提取提供
		if matches := providesRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processProvideLine(matches[1], currentPackage, &deps)
			continue
		}

		// 提取替换
		if matches := replacesRe.FindStringSubmatch(line); len(matches) > 1 {
			e.processReplaceLine(matches[1], currentPackage, &deps)
			continue
		}
	}

	// 处理最后一个继续行
	if continuationLine != "" {
		e.processDependencyLine(continuationLine, currentPackage, &deps)
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(ControlExtractorType, e.FilePath, err.Error())
	}

	return deps, nil
}

// processDependencyLine 处理依赖行
func (e *ControlExtractor) processDependencyLine(line, currentPackage string, deps *[]models.Dependency) {
	// 分割依赖项
	items := strings.Split(line, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		// 解析依赖项
		parts := strings.Fields(item)
		name := parts[0]
		var version string
		var operator string

		// 解析版本约束
		if len(parts) >= 3 && (parts[1] == ">=" || parts[1] == "<=" || parts[1] == "=" || parts[1] == ">" || parts[1] == "<") {
			operator = parts[1]
			version = parts[2]
		}

		dep := models.NewDependency(name)
		dep.Type = "dependency"
		dep.BuildSystem = "debian"
		dep.DetectedBy = "ControlExtractor"
		dep.ConfigFile = e.FilePath
		dep.ConfigFileType = "control"
		dep.Required = true

		if operator != "" && version != "" {
			dep.Constraints = append(dep.Constraints, models.VersionConstrain{
				Operator: operator,
				Version:  version,
			})
		}

		*deps = append(*deps, *dep)
	}
}

// processOptionalDependencyLine 处理可选依赖行
func (e *ControlExtractor) processOptionalDependencyLine(line, currentPackage, depType string, deps *[]models.Dependency) {
	items := strings.Split(line, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.Fields(item)
		name := parts[0]

		dep := models.NewDependency(name)
		dep.Type = depType
		dep.BuildSystem = "debian"
		dep.DetectedBy = "ControlExtractor"
		dep.ConfigFile = e.FilePath
		dep.ConfigFileType = "control"
		dep.Required = false
		dep.Optional = true

		*deps = append(*deps, *dep)
	}
}

// processConflictLine 处理冲突行
func (e *ControlExtractor) processConflictLine(line, currentPackage, conflictType string, deps *[]models.Dependency) {
	items := strings.Split(line, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.Fields(item)
		name := parts[0]

		dep := models.NewDependency(name)
		dep.Type = conflictType
		dep.BuildSystem = "debian"
		dep.DetectedBy = "ControlExtractor"
		dep.ConfigFile = e.FilePath
		dep.ConfigFileType = "control"
		dep.Required = false

		*deps = append(*deps, *dep)
	}
}

// processProvideLine 处理提供行
func (e *ControlExtractor) processProvideLine(line, currentPackage string, deps *[]models.Dependency) {
	items := strings.Split(line, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.Fields(item)
		name := parts[0]

		dep := models.NewDependency(name)
		dep.Type = "provides"
		dep.BuildSystem = "debian"
		dep.DetectedBy = "ControlExtractor"
		dep.ConfigFile = e.FilePath
		dep.ConfigFileType = "control"
		dep.Required = false

		*deps = append(*deps, *dep)
	}
}

// processReplaceLine 处理替换行
func (e *ControlExtractor) processReplaceLine(line, currentPackage string, deps *[]models.Dependency) {
	items := strings.Split(line, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.Fields(item)
		name := parts[0]

		dep := models.NewDependency(name)
		dep.Type = "replaces"
		dep.BuildSystem = "debian"
		dep.DetectedBy = "ControlExtractor"
		dep.ConfigFile = e.FilePath
		dep.ConfigFileType = "control"
		dep.Required = false

		*deps = append(*deps, *dep)
	}
}

// ControlExtractorFactory Control提取器工厂
type ControlExtractorFactory struct{}

// CreateExtractor 创建Control提取器
func (f *ControlExtractorFactory) CreateExtractor(path string) Extractor {
	return NewControlExtractor(path)
}

func init() {
	// 注册Control提取器
	RegisterExtractor(ControlExtractorType, &ControlExtractorFactory{})
}

/*
使用示例:

1. 创建Control提取器:
extractor := NewControlExtractor("control")

2. 提取依赖:
deps, err := extractor.Extract()
if err != nil {
    log.Printf("Failed to extract dependencies: %v\n", err)
    return
}

3. 处理依赖信息:
for _, dep := range deps {
    fmt.Printf("Found dependency: %s (%s)\n", dep.Name, dep.Type)
    if len(dep.Constraints) > 0 {
        fmt.Printf("  Version constraints:\n")
        for _, c := range dep.Constraints {
            fmt.Printf("    %s %s\n", c.Operator, c.Version)
        }
    }
}

示例control文件:
```
Source: mypackage
Section: utils
Priority: optional
Maintainer: John Doe <john@example.com>
Build-Depends: debhelper (>= 9),
               cmake (>= 3.10),
               libboost-dev (>= 1.65),
               libssl-dev
Standards-Version: 4.5.0

Package: mypackage
Architecture: any
Depends: ${shlibs:Depends},
         ${misc:Depends},
         libboost-system1.74.0 (>= 1.74.0),
         libssl1.1 (>= 1.1.0)
Recommends: python3
Suggests: documentation-viewer
Conflicts: oldpackage (<< 2.0)
Description: Example package
 This is an example package description.
 .
 It can span multiple lines.
```
*/ 