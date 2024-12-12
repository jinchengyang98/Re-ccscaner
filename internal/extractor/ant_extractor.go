package extractor

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
	"go.uber.org/zap"
)

// AntExtractor Ant构建系统提取器
type AntExtractor struct {
	logger *zap.Logger
}

// AntProject Ant项目配置
type AntProject struct {
	XMLName     xml.Name     `xml:"project"`
	Name        string       `xml:"name,attr"`
	BasePath    string       `xml:"basedir,attr"`
	Properties  []Property   `xml:"property"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
	Imports     []Import     `xml:"import"`
	Paths       []Path       `xml:"path"`
}

// Property Ant属性
type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	File  string `xml:"file,attr"`
}

// Dependency Ant依赖
type Dependency struct {
	Name     string `xml:"name,attr"`
	Version  string `xml:"version,attr"`
	Required bool   `xml:"required,attr"`
	Type     string `xml:"type,attr"`
}

// Import Ant导入
type Import struct {
	File string `xml:"file,attr"`
}

// Path Ant路径
type Path struct {
	ID        string    `xml:"id,attr"`
	Locations []string  `xml:"location"`
}

// NewAntExtractor 创建Ant提取器
func NewAntExtractor(logger *zap.Logger) *AntExtractor {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &AntExtractor{
		logger: logger,
	}
}

// Extract 提取依赖信息
func (e *AntExtractor) Extract(dir string) ([]models.Dependency, error) {
	e.logger.Info("Starting Ant dependency extraction", zap.String("dir", dir))

	// 查找build.xml文件
	buildFiles, err := filepath.Glob(filepath.Join(dir, "**/build.xml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find build.xml files: %w", err)
	}

	var allDeps []models.Dependency
	for _, buildFile := range buildFiles {
		// 解析build.xml
		deps, err := e.parseBuildFile(buildFile)
		if err != nil {
			e.logger.Error("Failed to parse build file",
				zap.String("file", buildFile),
				zap.Error(err))
			continue
		}
		allDeps = append(allDeps, deps...)
	}

	e.logger.Info("Completed Ant dependency extraction",
		zap.Int("total_deps", len(allDeps)))
	return allDeps, nil
}

// parseBuildFile 解析build.xml文件
func (e *AntExtractor) parseBuildFile(file string) ([]models.Dependency, error) {
	// 读取文件内容
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read build file: %w", err)
	}

	// 解析XML
	var project AntProject
	if err := xml.Unmarshal(content, &project); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var deps []models.Dependency

	// 处理直接声明的依赖
	for _, dep := range project.Dependencies {
		deps = append(deps, models.Dependency{
			Name:        dep.Name,
			Version:     dep.Version,
			Type:        dep.Type,
			Required:    dep.Required,
			BuildSystem: "ant",
			Source:      file,
		})
	}

	// 处理属性文件中的依赖
	for _, prop := range project.Properties {
		if prop.File != "" && strings.Contains(prop.File, "dependencies") {
			propDeps, err := e.parsePropertyFile(filepath.Join(filepath.Dir(file), prop.File))
			if err != nil {
				e.logger.Error("Failed to parse property file",
					zap.String("file", prop.File),
					zap.Error(err))
				continue
			}
			deps = append(deps, propDeps...)
		}
	}

	// 处理导入的构建文件
	for _, imp := range project.Imports {
		if imp.File != "" {
			impDeps, err := e.parseBuildFile(filepath.Join(filepath.Dir(file), imp.File))
			if err != nil {
				e.logger.Error("Failed to parse imported build file",
					zap.String("file", imp.File),
					zap.Error(err))
				continue
			}
			deps = append(deps, impDeps...)
		}
	}

	// 处理路径中的依赖
	for _, path := range project.Paths {
		for _, loc := range path.Locations {
			if strings.Contains(loc, ".jar") {
				name, version := parseJarLocation(loc)
				deps = append(deps, models.Dependency{
					Name:        name,
					Version:     version,
					Type:        "jar",
					Required:    true,
					BuildSystem: "ant",
					Source:      file,
				})
			}
		}
	}

	return deps, nil
}

// parsePropertyFile 解析属性文件
func (e *AntExtractor) parsePropertyFile(file string) ([]models.Dependency, error) {
	// 读取属性文件
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read property file: %w", err)
	}

	var deps []models.Dependency
	lines := strings.Split(string(content), "\n")

	// 解析每一行
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 查找依赖定义
		if strings.Contains(line, "dependency") {
			parts := strings.Split(line, "=")
			if len(parts) != 2 {
				continue
			}

			name, version := parseDependencyValue(parts[1])
			if name != "" {
				deps = append(deps, models.Dependency{
					Name:        name,
					Version:     version,
					Type:        "jar",
					Required:    true,
					BuildSystem: "ant",
					Source:      file,
				})
			}
		}
	}

	return deps, nil
}

// parseJarLocation 解析JAR文件路径
func parseJarLocation(loc string) (name, version string) {
	// 提取文件名
	base := filepath.Base(loc)
	if !strings.HasSuffix(base, ".jar") {
		return "", ""
	}

	// 移除.jar后缀
	base = strings.TrimSuffix(base, ".jar")

	// 尝试分离名称和版本
	parts := strings.Split(base, "-")
	if len(parts) < 2 {
		return base, ""
	}

	// 最后一部分通常是版本号
	version = parts[len(parts)-1]
	name = strings.Join(parts[:len(parts)-1], "-")

	return name, version
}

// parseDependencyValue 解析依赖值
func parseDependencyValue(value string) (name, version string) {
	value = strings.TrimSpace(value)

	// 检查是否包含版本信息
	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
	}

	// 尝试从文件名解析
	if strings.HasSuffix(value, ".jar") {
		return parseJarLocation(value)
	}

	return value, ""
}

/*
使用示例:

1. 创建提取器:
extractor := NewAntExtractor(logger)

2. 提取依赖:
deps, err := extractor.Extract("/path/to/project")
if err != nil {
    log.Fatal(err)
}

3. 处理依赖信息:
for _, dep := range deps {
    fmt.Printf("Found dependency: %s@%s\n", dep.Name, dep.Version)
}

注意事项:

1. 文件解析:
- 支持解析build.xml文件
- 支持解析属性文件
- 支持解析导入的构建文件
- 支持从路径中提取JAR依赖

2. 依赖识别:
- 从直接声明的dependencies标签中提取
- 从属性文件中提取
- 从JAR文件路径中提取
- 支持版本号解析

3. 错误处理:
- 文件不存在或无法读取时返回错误
- XML解析失败时返回错误
- 单个文件解析失败不影响其他文件的处理

4. 日志记录:
- 记录开始和完成的提取过程
- 记录解析失败的文件
- 记录找到的依赖数量
*/ 