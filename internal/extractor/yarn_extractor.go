package extractor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// YarnExtractor Yarn包管理器提取器
type YarnExtractor struct {
	logger *zap.Logger
}

// YarnLockEntry yarn.lock文件中的依赖项
type YarnLockEntry struct {
	Version     string            `yaml:"version"`
	Resolution  string            `yaml:"resolution"`
	Integrity   string            `yaml:"integrity"`
	Dependencies map[string]string `yaml:"dependencies"`
	OptionalDependencies map[string]string `yaml:"optionalDependencies"`
}

// YarnLock yarn.lock文件结构
type YarnLock struct {
	Type         string                    `yaml:"__metadata"`
	Dependencies map[string]YarnLockEntry  `yaml:",inline"`
}

// YarnRC .yarnrc.yml文件结构
type YarnRC struct {
	NodeLinker      string   `yaml:"nodeLinker"`
	PackageExtensions map[string]map[string]interface{} `yaml:"packageExtensions"`
	Plugins        []string `yaml:"plugins"`
	YarnPath       string   `yaml:"yarnPath"`
}

// NewYarnExtractor 创建Yarn提取器
func NewYarnExtractor(logger *zap.Logger) *YarnExtractor {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &YarnExtractor{
		logger: logger,
	}
}

// Extract 提取依赖信息
func (e *YarnExtractor) Extract(dir string) ([]models.Dependency, error) {
	e.logger.Info("Starting Yarn dependency extraction", zap.String("dir", dir))

	// 查找package.json文件
	packageFiles, err := filepath.Glob(filepath.Join(dir, "**/package.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to find package.json files: %w", err)
	}

	var allDeps []models.Dependency
	for _, packageFile := range packageFiles {
		// 跳过node_modules目录中的package.json
		if strings.Contains(packageFile, "node_modules") {
			continue
		}

		// 解析package.json
		deps, err := e.parsePackageFile(packageFile)
		if err != nil {
			e.logger.Error("Failed to parse package file",
				zap.String("file", packageFile),
				zap.Error(err))
			continue
		}
		allDeps = append(allDeps, deps...)

		// 解析yarn.lock
		lockDeps, err := e.parseLockfile(filepath.Join(filepath.Dir(packageFile), "yarn.lock"))
		if err != nil {
			e.logger.Warn("Failed to parse lockfile",
				zap.String("file", packageFile),
				zap.Error(err))
		} else {
			allDeps = append(allDeps, lockDeps...)
		}

		// 解析.yarnrc.yml
		if err := e.parseYarnRC(filepath.Join(filepath.Dir(packageFile), ".yarnrc.yml")); err != nil {
			e.logger.Warn("Failed to parse .yarnrc.yml",
				zap.String("file", packageFile),
				zap.Error(err))
		}
	}

	e.logger.Info("Completed Yarn dependency extraction",
		zap.Int("total_deps", len(allDeps)))
	return allDeps, nil
}

// parsePackageFile 解析package.json文件
func (e *YarnExtractor) parsePackageFile(file string) ([]models.Dependency, error) {
	// 读取文件内容
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read package file: %w", err)
	}

	// 解析JSON
	var pkg PackageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var deps []models.Dependency

	// 处理生产依赖
	for name, version := range pkg.Dependencies {
		deps = append(deps, models.Dependency{
			Name:        name,
			Version:     cleanVersion(version),
			Type:        "production",
			Required:    true,
			BuildSystem: "yarn",
			Source:      file,
		})
	}

	// 处理开发依赖
	for name, version := range pkg.DevDependencies {
		deps = append(deps, models.Dependency{
			Name:        name,
			Version:     cleanVersion(version),
			Type:        "development",
			Required:    true,
			BuildSystem: "yarn",
			Source:      file,
		})
	}

	// 处理对等依赖
	for name, version := range pkg.PeerDependencies {
		deps = append(deps, models.Dependency{
			Name:        name,
			Version:     cleanVersion(version),
			Type:        "peer",
			Required:    true,
			BuildSystem: "yarn",
			Source:      file,
		})
	}

	// 处理可选依赖
	for name, version := range pkg.OptionalDependencies {
		deps = append(deps, models.Dependency{
			Name:        name,
			Version:     cleanVersion(version),
			Type:        "optional",
			Required:    false,
			BuildSystem: "yarn",
			Source:      file,
		})
	}

	// 处理工作区
	if len(pkg.Workspaces) > 0 {
		for _, workspace := range pkg.Workspaces {
			// 支持glob模式
			matches, err := filepath.Glob(filepath.Join(filepath.Dir(file), workspace))
			if err != nil {
				e.logger.Error("Failed to resolve workspace pattern",
					zap.String("pattern", workspace),
					zap.Error(err))
				continue
			}

			for _, match := range matches {
				workspaceFile := filepath.Join(match, "package.json")
				if _, err := ioutil.Stat(workspaceFile); err == nil {
					workspaceDeps, err := e.parsePackageFile(workspaceFile)
					if err != nil {
						e.logger.Error("Failed to parse workspace package file",
							zap.String("file", workspaceFile),
							zap.Error(err))
						continue
					}
					deps = append(deps, workspaceDeps...)
				}
			}
		}
	}

	return deps, nil
}

// parseLockfile 解析yarn.lock文件
func (e *YarnExtractor) parseLockfile(file string) ([]models.Dependency, error) {
	// 读取文件内容
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	// 解析YAML
	var lock YarnLock
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var deps []models.Dependency

	// 处理锁定的依赖
	for name, entry := range lock.Dependencies {
		// 解析依赖标识符
		parts := strings.Split(name, "@")
		if len(parts) < 2 {
			continue
		}

		dep := models.Dependency{
			Name:        parts[0],
			Version:     entry.Version,
			Type:        "locked",
			Required:    true,
			BuildSystem: "yarn",
			Source:      file,
		}

		// 添加解析URL
		if entry.Resolution != "" {
			dep.Source = fmt.Sprintf("%s (%s)", file, entry.Resolution)
		}

		// 添加完整性校验
		if entry.Integrity != "" {
			dep.Source = fmt.Sprintf("%s [%s]", dep.Source, entry.Integrity)
		}

		// 处理子依赖
		var subDeps []models.Dependency
		for subName, subVersion := range entry.Dependencies {
			subDeps = append(subDeps, models.Dependency{
				Name:        subName,
				Version:     subVersion,
				Type:        "transitive",
				Required:    true,
				BuildSystem: "yarn",
				Source:      file,
			})
		}

		// 处理可选子依赖
		for subName, subVersion := range entry.OptionalDependencies {
			subDeps = append(subDeps, models.Dependency{
				Name:        subName,
				Version:     subVersion,
				Type:        "transitive",
				Required:    false,
				BuildSystem: "yarn",
				Source:      file,
			})
		}

		dep.Dependencies = subDeps
		deps = append(deps, dep)
	}

	return deps, nil
}

// parseYarnRC 解析.yarnrc.yml文件
func (e *YarnExtractor) parseYarnRC(file string) error {
	// 读取文件内容
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read .yarnrc.yml: %w", err)
	}

	// 解析YAML
	var rc YarnRC
	if err := yaml.Unmarshal(content, &rc); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// 记录配置信息
	e.logger.Info("Yarn configuration",
		zap.String("nodeLinker", rc.NodeLinker),
		zap.Int("plugins", len(rc.Plugins)),
		zap.Int("packageExtensions", len(rc.PackageExtensions)))

	return nil
}

/*
使用示例:

1. 创建提取器:
extractor := NewYarnExtractor(logger)

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
- 支持解析package.json文件
- 支持解析yarn.lock文件
- 支持解析.yarnrc.yml文件
- 支持工作区(workspaces)
- 跳过node_modules目录

2. 依赖类型:
- 生产依赖(dependencies)
- 开发依赖(devDependencies)
- 对等依赖(peerDependencies)
- 可选依赖(optionalDependencies)
- 锁定依赖(从yarn.lock)
- 传递依赖(子依赖)

3. 版本处理:
- 清理版本范围标记
- 支持URL和文件路径
- 支持git URL
- 支持标签(latest, next等)

4. 错误处理:
- 文件不存在或无法读取时返回错误
- YAML/JSON解析失败时返回错误
- 单个文件解析失败不影响其他文件的处理

5. 日志记录:
- 记录开始和完成的提取过程
- 记录解析失败的文件
- 记录找到的依赖数量
- 记录Yarn配置信息
*/ 