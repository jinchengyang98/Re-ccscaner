package extractor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/lkpsg/ccscanner/pkg/models"
)

// SubmoduleExtractor Git子模块依赖提取器
type SubmoduleExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewSubmoduleExtractor 创建Git子模块提取器
func NewSubmoduleExtractor(path string) *SubmoduleExtractor {
	return &SubmoduleExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取Git子模块依赖
func (e *SubmoduleExtractor) Extract() ([]models.Dependency, error) {
	// 获取.gitmodules文件路径
	gitmodulesPath := filepath.Join(filepath.Dir(e.FilePath), ".gitmodules")
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		return nil, NewExtractorError(SubmoduleExtractorType, e.FilePath, ".gitmodules file not found")
	}

	// 读取.gitmodules文件
	file, err := os.Open(gitmodulesPath)
	if err != nil {
		return nil, NewExtractorError(SubmoduleExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	// 正则表达式
	submoduleRe := regexp.MustCompile(`\[submodule "([^"]+)"\]`)
	pathRe := regexp.MustCompile(`\s*path\s*=\s*(.+)`)
	urlRe := regexp.MustCompile(`\s*url\s*=\s*(.+)`)
	branchRe := regexp.MustCompile(`\s*branch\s*=\s*(.+)`)

	var currentDep *models.Dependency
	lineNum := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum++

		// 忽略空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查子模块声明
		if matches := submoduleRe.FindStringSubmatch(line); len(matches) > 1 {
			// 保存前一个子模块
			if currentDep != nil {
				deps = append(deps, *currentDep)
			}

			// 创建新的子模块依赖
			currentDep = models.NewDependency(matches[1])
			currentDep.Type = "submodule"
			currentDep.BuildSystem = "git"
			currentDep.DetectedBy = "SubmoduleExtractor"
			currentDep.ConfigFile = gitmodulesPath
			currentDep.ConfigFileType = ".gitmodules"
			continue
		}

		if currentDep == nil {
			continue
		}

		// 提取路径
		if matches := pathRe.FindStringSubmatch(line); len(matches) > 1 {
			path := strings.TrimSpace(matches[1])
			currentDep.Description = fmt.Sprintf("Git submodule at %s", path)
			
			// 尝试获取子模块的提交信息
			if err := e.extractSubmoduleInfo(path, currentDep); err != nil {
				// 记录错误但继续处理
				currentDep.Description += fmt.Sprintf(" (Error: %v)", err)
			}
			continue
		}

		// 提取URL
		if matches := urlRe.FindStringSubmatch(line); len(matches) > 1 {
			url := strings.TrimSpace(matches[1])
			currentDep.Repository = url
			
			// 从URL提取来源信息
			if strings.Contains(url, "github.com") {
				currentDep.Source = "github"
			} else if strings.Contains(url, "gitlab.com") {
				currentDep.Source = "gitlab"
			} else if strings.Contains(url, "bitbucket.org") {
				currentDep.Source = "bitbucket"
			}
			continue
		}

		// 提取分支
		if matches := branchRe.FindStringSubmatch(line); len(matches) > 1 {
			currentDep.Branch = strings.TrimSpace(matches[1])
			continue
		}
	}

	// 保存最后一个子模块
	if currentDep != nil {
		deps = append(deps, *currentDep)
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(SubmoduleExtractorType, e.FilePath, err.Error())
	}

	return deps, nil
}

// extractSubmoduleInfo 提取子模块的Git信息
func (e *SubmoduleExtractor) extractSubmoduleInfo(path string, dep *models.Dependency) error {
	// 获取子模块的完整路径
	fullPath := filepath.Join(filepath.Dir(e.FilePath), path)

	// 打开Git仓库
	repo, err := git.PlainOpen(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %v", err)
	}

	// 获取HEAD引用
	ref, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %v", err)
	}

	// 设置提交哈希
	dep.Commit = ref.Hash().String()

	// 获取远程信息
	remotes, err := repo.Remotes()
	if err != nil {
		return fmt.Errorf("failed to get remotes: %v", err)
	}

	// 从远程URL更新仓库信息
	if len(remotes) > 0 {
		urls := remotes[0].Config().URLs
		if len(urls) > 0 {
			dep.Repository = urls[0]
		}
	}

	return nil
}

// SubmoduleExtractorFactory Git子模块提取器工厂
type SubmoduleExtractorFactory struct{}

// CreateExtractor 创建Git子模块提取器
func (f *SubmoduleExtractorFactory) CreateExtractor(path string) Extractor {
	return NewSubmoduleExtractor(path)
}

func init() {
	// 注册Git子模块提取器
	RegisterExtractor(SubmoduleExtractorType, &SubmoduleExtractorFactory{})
}

/*
使用示例:

1. 创建Git子模块提取器:
extractor := NewSubmoduleExtractor("/path/to/repo")

2. 提取依赖:
deps, err := extractor.Extract()
if err != nil {
    log.Printf("Failed to extract dependencies: %v\n", err)
    return
}

3. 处理依赖信息:
for _, dep := range deps {
    fmt.Printf("Found submodule: %s\n", dep.Name)
    fmt.Printf("  Repository: %s\n", dep.Repository)
    fmt.Printf("  Branch: %s\n", dep.Branch)
    fmt.Printf("  Commit: %s\n", dep.Commit)
    fmt.Printf("  Description: %s\n", dep.Description)
}

示例.gitmodules文件:
```
[submodule "libs/googletest"]
    path = libs/googletest
    url = https://github.com/google/googletest.git
    branch = main

[submodule "libs/json"]
    path = libs/json
    url = https://github.com/nlohmann/json.git
    branch = develop

[submodule "docs/theme"]
    path = docs/theme
    url = https://github.com/pages-themes/cayman.git
```
*/ 