package extractor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// ConanExtractor Conan依赖提取器
type ConanExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewConanExtractor 创建Conan提取器
func NewConanExtractor(path string) *ConanExtractor {
	return &ConanExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取Conan依赖
func (e *ConanExtractor) Extract() ([]models.Dependency, error) {
	deps := make([]models.Dependency, 0)

	// 根据文件类型选择提取方法
	switch filepath.Base(e.FilePath) {
	case "conanfile.txt":
		return e.extractFromTxt()
	case "conanfile.py":
		return e.extractFromPy()
	case "conaninfo.txt":
		return e.extractFromInfo()
	default:
		return nil, NewExtractorError(ConanExtractorType, e.FilePath, "unsupported file type")
	}
}

// extractFromTxt 从conanfile.txt提取依赖
func (e *ConanExtractor) extractFromTxt() ([]models.Dependency, error) {
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(ConanExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	var inRequiresSection bool
	requireRe := regexp.MustCompile(`^(\S+)/(\S+)(@\S+)?$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 忽略空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查节标记
		if line == "[requires]" {
			inRequiresSection = true
			continue
		} else if strings.HasPrefix(line, "[") {
			inRequiresSection = false
			continue
		}

		// 提取依赖
		if inRequiresSection {
			if matches := requireRe.FindStringSubmatch(line); len(matches) > 1 {
				name := matches[1]
				version := matches[2]
				channel := ""
				if len(matches) > 3 && matches[3] != "" {
					channel = matches[3][1:] // 去掉@前缀
				}

				dep := models.NewDependency(name)
				dep.Version = version
				dep.Type = "library"
				dep.BuildSystem = "conan"
				dep.DetectedBy = "ConanExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "conanfile.txt"
				if channel != "" {
					dep.Source = channel
				}

				deps = append(deps, *dep)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(ConanExtractorType, e.FilePath, err.Error())
	}

	return deps, nil
}

// extractFromPy 从conanfile.py提取依赖
func (e *ConanExtractor) extractFromPy() ([]models.Dependency, error) {
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(ConanExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	requiresRe := regexp.MustCompile(`requires\s*=\s*["']([^"']+)["']`)
	requireRe := regexp.MustCompile(`self\.requires\(["']([^"']+)["']\)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 忽略注释
		if strings.HasPrefix(line, "#") {
			continue
		}

		// 提取requires属性
		if matches := requiresRe.FindStringSubmatch(line); len(matches) > 1 {
			reqs := strings.Split(matches[1], ",")
			for _, req := range reqs {
				req = strings.TrimSpace(req)
				parts := strings.Split(req, "/")
				if len(parts) >= 2 {
					dep := models.NewDependency(parts[0])
					dep.Version = parts[1]
					dep.Type = "library"
					dep.BuildSystem = "conan"
					dep.DetectedBy = "ConanExtractor"
					dep.ConfigFile = e.FilePath
					dep.ConfigFileType = "conanfile.py"
					if len(parts) > 2 {
						dep.Source = parts[2]
					}
					deps = append(deps, *dep)
				}
			}
		}

		// 提取self.requires()调用
		if matches := requireRe.FindStringSubmatch(line); len(matches) > 1 {
			req := strings.TrimSpace(matches[1])
			parts := strings.Split(req, "/")
			if len(parts) >= 2 {
				dep := models.NewDependency(parts[0])
				dep.Version = parts[1]
				dep.Type = "library"
				dep.BuildSystem = "conan"
				dep.DetectedBy = "ConanExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "conanfile.py"
				if len(parts) > 2 {
					dep.Source = parts[2]
				}
				deps = append(deps, *dep)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(ConanExtractorType, e.FilePath, err.Error())
	}

	return deps, nil
}

// extractFromInfo 从conaninfo.txt提取依赖
func (e *ConanExtractor) extractFromInfo() ([]models.Dependency, error) {
	file, err := os.Open(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(ConanExtractorType, e.FilePath, err.Error())
	}
	defer file.Close()

	deps := make([]models.Dependency, 0)
	scanner := bufio.NewScanner(file)

	var inRequiresSection bool
	requireRe := regexp.MustCompile(`^\s*(\S+)/(\S+)(@\S+)?#\S+$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 忽略空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查节标记
		if line == "[requires]" {
			inRequiresSection = true
			continue
		} else if strings.HasPrefix(line, "[") {
			inRequiresSection = false
			continue
		}

		// 提取依赖
		if inRequiresSection {
			if matches := requireRe.FindStringSubmatch(line); len(matches) > 1 {
				name := matches[1]
				version := matches[2]
				channel := ""
				if len(matches) > 3 && matches[3] != "" {
					channel = matches[3][1:] // 去掉@前缀
				}

				dep := models.NewDependency(name)
				dep.Version = version
				dep.Type = "library"
				dep.BuildSystem = "conan"
				dep.DetectedBy = "ConanExtractor"
				dep.ConfigFile = e.FilePath
				dep.ConfigFileType = "conaninfo.txt"
				if channel != "" {
					dep.Source = channel
				}

				deps = append(deps, *dep)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, NewExtractorError(ConanExtractorType, e.FilePath, err.Error())
	}

	return deps, nil
}

// ConanExtractorFactory Conan提取器工厂
type ConanExtractorFactory struct{}

// CreateExtractor 创建Conan提取器
func (f *ConanExtractorFactory) CreateExtractor(path string) Extractor {
	return NewConanExtractor(path)
}

func init() {
	// 注册Conan提取器
	RegisterExtractor(ConanExtractorType, &ConanExtractorFactory{})
}

/*
使用示例:

1. 创建Conan提取器:
extractor := NewConanExtractor("conanfile.txt")

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
    fmt.Printf("Found dependency: %s/%s (%s)\n", dep.Name, dep.Version, dep.Source)
}

示例conanfile.txt文件:
```txt
[requires]
boost/1.76.0
openssl/1.1.1k@conan/stable
zlib/1.2.11

[generators]
cmake
```

示例conanfile.py文件:
```python
from conans import ConanFile, CMake

class MyLibConan(ConanFile):
    name = "mylib"
    version = "1.0.0"
    requires = "boost/1.76.0, openssl/1.1.1k@conan/stable"
    
    def requirements(self):
        self.requires("zlib/1.2.11")
```

示例conaninfo.txt文件:
```txt
[requires]
boost/1.76.0@conan/stable#0123456789
openssl/1.1.1k@conan/stable#9876543210
zlib/1.2.11#abcdef0123
```
*/ 