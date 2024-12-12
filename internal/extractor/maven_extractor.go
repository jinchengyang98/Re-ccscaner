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

// MavenExtractor Maven构建系统提取器
type MavenExtractor struct {
	logger *zap.Logger
}

// MavenProject Maven项目配置
type MavenProject struct {
	XMLName      xml.Name        `xml:"project"`
	GroupID      string          `xml:"groupId"`
	ArtifactID   string          `xml:"artifactId"`
	Version      string          `xml:"version"`
	Parent       MavenParent     `xml:"parent"`
	Properties   MavenProperties `xml:"properties"`
	Dependencies []MavenDependency `xml:"dependencies>dependency"`
	Modules      []string        `xml:"modules>module"`
	Profiles     []MavenProfile  `xml:"profiles>profile"`
}

// MavenParent Maven父项目
type MavenParent struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

// MavenProperties Maven属性
type MavenProperties struct {
	Properties map[string]string `xml:",any"`
}

// MavenDependency Maven依赖
type MavenDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   bool   `xml:"optional"`
	Type       string `xml:"type"`
	Classifier string `xml:"classifier"`
	Exclusions []MavenExclusion `xml:"exclusions>exclusion"`
}

// MavenExclusion Maven排除项
type MavenExclusion struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
}

// MavenProfile Maven配置文件
type MavenProfile struct {
	ID           string           `xml:"id"`
	Dependencies []MavenDependency `xml:"dependencies>dependency"`
}

// NewMavenExtractor 创建Maven提取器
func NewMavenExtractor(logger *zap.Logger) *MavenExtractor {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &MavenExtractor{
		logger: logger,
	}
}

// Extract 提取依赖信息
func (e *MavenExtractor) Extract(dir string) ([]models.Dependency, error) {
	e.logger.Info("Starting Maven dependency extraction", zap.String("dir", dir))

	// 查找pom.xml文件
	pomFiles, err := filepath.Glob(filepath.Join(dir, "**/pom.xml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find pom.xml files: %w", err)
	}

	var allDeps []models.Dependency
	for _, pomFile := range pomFiles {
		// 解析pom.xml
		deps, err := e.parsePomFile(pomFile)
		if err != nil {
			e.logger.Error("Failed to parse POM file",
				zap.String("file", pomFile),
				zap.Error(err))
			continue
		}
		allDeps = append(allDeps, deps...)
	}

	e.logger.Info("Completed Maven dependency extraction",
		zap.Int("total_deps", len(allDeps)))
	return allDeps, nil
}

// parsePomFile 解析pom.xml文件
func (e *MavenExtractor) parsePomFile(file string) ([]models.Dependency, error) {
	// 读取文件内容
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read POM file: %w", err)
	}

	// 解析XML
	var project MavenProject
	if err := xml.Unmarshal(content, &project); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var deps []models.Dependency

	// 处理父项目依赖
	if project.Parent.GroupID != "" {
		deps = append(deps, models.Dependency{
			Name:        fmt.Sprintf("%s:%s", project.Parent.GroupID, project.Parent.ArtifactID),
			Version:     project.Parent.Version,
			Type:        "parent",
			Required:    true,
			BuildSystem: "maven",
			Source:      file,
		})
	}

	// 处理直接依赖
	for _, dep := range project.Dependencies {
		deps = append(deps, convertMavenDependency(dep, file))
	}

	// 处理配置文件中的依赖
	for _, profile := range project.Profiles {
		for _, dep := range profile.Dependencies {
			mavenDep := convertMavenDependency(dep, file)
			mavenDep.Source = fmt.Sprintf("%s (profile: %s)", file, profile.ID)
			deps = append(deps, mavenDep)
		}
	}

	// 处理子模块
	for _, module := range project.Modules {
		modulePath := filepath.Join(filepath.Dir(file), module, "pom.xml")
		moduleDeps, err := e.parsePomFile(modulePath)
		if err != nil {
			e.logger.Error("Failed to parse module POM file",
				zap.String("file", modulePath),
				zap.Error(err))
			continue
		}
		deps = append(deps, moduleDeps...)
	}

	return deps, nil
}

// convertMavenDependency 转换Maven依赖为通用依赖
func convertMavenDependency(dep MavenDependency, source string) models.Dependency {
	// 确定依赖类型
	depType := "compile"
	if dep.Type != "" {
		depType = dep.Type
	} else if dep.Scope != "" {
		depType = dep.Scope
	}

	// 构建依赖名称
	name := fmt.Sprintf("%s:%s", dep.GroupID, dep.ArtifactID)
	if dep.Classifier != "" {
		name = fmt.Sprintf("%s:%s", name, dep.Classifier)
	}

	// 处理版本号中的属性引用
	version := dep.Version
	if strings.HasPrefix(version, "${") && strings.HasSuffix(version, "}") {
		// TODO: 实现属性解析
		version = strings.TrimSuffix(strings.TrimPrefix(version, "${"), "}")
	}

	// 构建排除项列表
	var conflicts []models.Dependency
	for _, excl := range dep.Exclusions {
		conflicts = append(conflicts, models.Dependency{
			Name:        fmt.Sprintf("%s:%s", excl.GroupID, excl.ArtifactID),
			BuildSystem: "maven",
		})
	}

	return models.Dependency{
		Name:        name,
		Version:     version,
		Type:        depType,
		Required:    !dep.Optional,
		BuildSystem: "maven",
		Source:      source,
		Conflicts:   conflicts,
	}
}

/*
使用示例:

1. 创建提取器:
extractor := NewMavenExtractor(logger)

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
- 支持解析pom.xml文件
- 支持解析父项目依赖
- 支持解析子模块依赖
- 支持解析配置文件中的依赖

2. 依赖识别:
- 支持groupId:artifactId格式
- 支持classifier
- 支持scope和type
- 支持optional标记
- 支持exclusions

3. 错误处理:
- 文件不存在或无法读取时返回错误
- XML解析失败时返回错误
- 单个文件解析失败不影响其他文件的处理

4. 日志记录:
- 记录开始和完成的提取过程
- 记录解析失败的文件
- 记录找到的依赖数量

5. 待改进:
- 实现属性解析
- 支持更多的Maven特性
- 优化性能
*/ 