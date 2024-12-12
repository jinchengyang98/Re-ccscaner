package extractor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
	"go.uber.org/zap"
)

// NPMExtractor NPM包管理器提取器
type NPMExtractor struct {
	logger *zap.Logger
}

// PackageJSON package.json文件结构
type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	Workspaces     []string          `json:"workspaces"`
	Private        bool              `json:"private"`
}

// LockfileEntry package-lock.json文件中的依赖项
type LockfileEntry struct {
	Version      string                     `json:"version"`
	Resolved     string                     `json:"resolved"`
	Integrity    string                     `json:"integrity"`
	Dev          bool                       `json:"dev"`
	Optional     bool                       `json:"optional"`
	Dependencies map[string]string          `json:"dependencies"`
	Requires     map[string]string          `json:"requires"`
}

// PackageLock package-lock.json文件结构
type PackageLock struct {
	Name         string                     `json:"name"`
	Version      string                     `json:"version"`
	LockfileVersion int                     `json:"lockfileVersion"`
	Dependencies map[string]LockfileEntry   `json:"dependencies"`
}

// NewNPMExtractor 创建NPM提取器
func NewNPMExtractor(logger *zap.Logger) *NPMExtractor {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &NPMExtractor{
		logger: logger,
	}
}

// Extract 提取依赖信息
func (e *NPMExtractor) Extract(dir string) ([]models.Dependency, error) {
	e.logger.Info("Starting NPM dependency extraction", zap.String("dir", dir))

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

		// 解析package-lock.json
		lockDeps, err := e.parseLockfile(filepath.Join(filepath.Dir(packageFile), "package-lock.json"))
		if err != nil {
			e.logger.Warn("Failed to parse lockfile",
				zap.String("file", packageFile),
				zap.Error(err))
		} else {
			allDeps = append(allDeps, lockDeps...)
		}
	}

	e.logger.Info("Completed NPM dependency extraction",
		zap.Int("total_deps", len(allDeps)))
	return allDeps, nil
}

// parsePackageFile 解析package.json文件
func (e *NPMExtractor) parsePackageFile(file string) ([]models.Dependency, error) {
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
			BuildSystem: "npm",
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
			BuildSystem: "npm",
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
			BuildSystem: "npm",
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
			BuildSystem: "npm",
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

// parseLockfile 解析package-lock.json文件
func (e *NPMExtractor) parseLockfile(file string) ([]models.Dependency, error) {
	// 读取文件内容
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	// 解析JSON
	var lock PackageLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var deps []models.Dependency

	// 处理锁定的依赖
	for name, entry := range lock.Dependencies {
		dep := models.Dependency{
			Name:        name,
			Version:     entry.Version,
			Type:        "locked",
			Required:    !entry.Optional,
			BuildSystem: "npm",
			Source:      file,
		}

		// 添加解析URL
		if entry.Resolved != "" {
			dep.Source = fmt.Sprintf("%s (%s)", file, entry.Resolved)
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
				BuildSystem: "npm",
				Source:      file,
			})
		}
		dep.Dependencies = subDeps

		deps = append(deps, dep)
	}

	return deps, nil
}

// cleanVersion 清理版本号
func cleanVersion(version string) string {
	// 移除版本范围标记
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, ">")
	version = strings.TrimPrefix(version, "<=")
	version = strings.TrimPrefix(version, "<")
	version = strings.TrimPrefix(version, "=")

	// 处理URL和文件路径
	if strings.Contains(version, "://") || strings.HasPrefix(version, "file:") {
		return version
	}

	// 处理git URL
	if strings.Contains(version, "git") || strings.Contains(version, "github") {
		return version
	}

	// 处理标签
	if strings.HasPrefix(version, "latest") || strings.HasPrefix(version, "next") {
		return version
	}

	// 移除空格
	return strings.TrimSpace(version)
}

/*
使用示例:

1. 创建提取器:
extractor := NewNPMExtractor(logger)

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
- 支持解析package-lock.json文件
- 支持工作区(workspaces)
- 跳过node_modules目录

2. 依赖类型:
- 生产依赖(dependencies)
- 开发依赖(devDependencies)
- 对等依赖(peerDependencies)
- 可选依赖(optionalDependencies)
- 锁定依赖(从package-lock.json)
- 传递依赖(子依赖)

3. 版本处理:
- 清理版本范围标记
- 支持URL和文件路径
- 支持git URL
- 支持标签(latest, next等)

4. 错误处理:
- 文件不存在或无法读取时返回错误
- JSON解析失败时返回错误
- 单个文件解析失败不影响其他文件的处理

5. 日志记录:
- 记录开始和完成的提取过程
- 记录解析失败的文件
- 记录找到的依赖数量
*/ 