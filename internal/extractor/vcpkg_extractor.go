package extractor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// VcpkgDependency vcpkg.json中的依赖定义
type VcpkgDependency struct {
	Name         string            `json:"name"`
	Version      string            `json:"version-string,omitempty"`
	VersionDate  string            `json:"version-date,omitempty"`
	VersionSemver string          `json:"version-semver,omitempty"`
	Port         string            `json:"port-version,omitempty"`
	Features     []string          `json:"features,omitempty"`
	Default      []string          `json:"default-features,omitempty"`
	Platform     string            `json:"platform,omitempty"`
	Overrides    []VcpkgOverride   `json:"overrides,omitempty"`
}

// VcpkgOverride vcpkg.json中的依赖覆盖定义
type VcpkgOverride struct {
	Name         string `json:"name"`
	Version      string `json:"version-string,omitempty"`
	VersionDate  string `json:"version-date,omitempty"`
	VersionSemver string `json:"version-semver,omitempty"`
	Port         string `json:"port-version,omitempty"`
}

// VcpkgManifest vcpkg.json清单文件
type VcpkgManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version-string,omitempty"`
	Dependencies []VcpkgDependency `json:"dependencies"`
	Features     map[string]struct {
		Description  string            `json:"description"`
		Dependencies []VcpkgDependency `json:"dependencies,omitempty"`
	} `json:"features,omitempty"`
	DefaultFeatures []string          `json:"default-features,omitempty"`
	Overrides      []VcpkgOverride    `json:"overrides,omitempty"`
}

// VcpkgExtractor Vcpkg依赖提取器
type VcpkgExtractor struct {
	BaseExtractor
	config ExtractorConfig
}

// NewVcpkgExtractor 创建Vcpkg提取器
func NewVcpkgExtractor(path string) *VcpkgExtractor {
	return &VcpkgExtractor{
		BaseExtractor: NewBaseExtractor(path),
		config:       DefaultConfig,
	}
}

// Extract 提取Vcpkg依赖
func (e *VcpkgExtractor) Extract() ([]models.Dependency, error) {
	// 读取vcpkg.json文件
	data, err := os.ReadFile(e.FilePath)
	if err != nil {
		return nil, NewExtractorError(VcpkgExtractorType, e.FilePath, err.Error())
	}

	// 解析JSON
	var manifest VcpkgManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, NewExtractorError(VcpkgExtractorType, e.FilePath, fmt.Sprintf("failed to parse vcpkg.json: %v", err))
	}

	deps := make([]models.Dependency, 0)

	// 处理主要依赖
	for _, vcpkgDep := range manifest.Dependencies {
		dep := e.convertVcpkgDependency(vcpkgDep)
		deps = append(deps, *dep)
	}

	// 处理特性依赖
	for featureName, feature := range manifest.Features {
		for _, vcpkgDep := range feature.Dependencies {
			dep := e.convertVcpkgDependency(vcpkgDep)
			dep.Type = "feature"
			dep.Optional = true
			dep.Description = fmt.Sprintf("Feature: %s - %s", featureName, feature.Description)
			deps = append(deps, *dep)
		}
	}

	// 处理覆盖
	for _, override := range manifest.Overrides {
		dep := e.convertVcpkgOverride(override)
		deps = append(deps, *dep)
	}

	return deps, nil
}

// convertVcpkgDependency 转换Vcpkg依赖为通用依赖模型
func (e *VcpkgExtractor) convertVcpkgDependency(vcpkgDep VcpkgDependency) *models.Dependency {
	dep := models.NewDependency(vcpkgDep.Name)
	
	// 设置版本信息
	if vcpkgDep.Version != "" {
		dep.Version = vcpkgDep.Version
	} else if vcpkgDep.VersionSemver != "" {
		dep.Version = vcpkgDep.VersionSemver
	} else if vcpkgDep.VersionDate != "" {
		dep.Version = vcpkgDep.VersionDate
	}

	// 设置其他信息
	dep.Type = "library"
	dep.BuildSystem = "vcpkg"
	dep.DetectedBy = "VcpkgExtractor"
	dep.ConfigFile = e.FilePath
	dep.ConfigFileType = "vcpkg.json"

	// 设置特性信息
	if len(vcpkgDep.Features) > 0 {
		dep.BuildFlags = vcpkgDep.Features
	}
	if len(vcpkgDep.Default) > 0 {
		dep.BuildFlags = append(dep.BuildFlags, vcpkgDep.Default...)
	}

	// 设置平台信息
	if vcpkgDep.Platform != "" {
		dep.BuildFlags = append(dep.BuildFlags, fmt.Sprintf("platform:%s", vcpkgDep.Platform))
	}

	// 设置端口版本
	if vcpkgDep.Port != "" {
		dep.BuildFlags = append(dep.BuildFlags, fmt.Sprintf("port:%s", vcpkgDep.Port))
	}

	return dep
}

// convertVcpkgOverride 转换Vcpkg覆盖为通用依赖模型
func (e *VcpkgExtractor) convertVcpkgOverride(override VcpkgOverride) *models.Dependency {
	dep := models.NewDependency(override.Name)
	
	// 设置版本信息
	if override.Version != "" {
		dep.Version = override.Version
	} else if override.VersionSemver != "" {
		dep.Version = override.VersionSemver
	} else if override.VersionDate != "" {
		dep.Version = override.VersionDate
	}

	// 设置其他信息
	dep.Type = "override"
	dep.BuildSystem = "vcpkg"
	dep.DetectedBy = "VcpkgExtractor"
	dep.ConfigFile = e.FilePath
	dep.ConfigFileType = "vcpkg.json"

	// 设置端口版本
	if override.Port != "" {
		dep.BuildFlags = append(dep.BuildFlags, fmt.Sprintf("port:%s", override.Port))
	}

	return dep
}

// VcpkgExtractorFactory Vcpkg提取器工厂
type VcpkgExtractorFactory struct{}

// CreateExtractor 创建Vcpkg提取器
func (f *VcpkgExtractorFactory) CreateExtractor(path string) Extractor {
	return NewVcpkgExtractor(path)
}

func init() {
	// 注册Vcpkg提取器
	RegisterExtractor(VcpkgExtractorType, &VcpkgExtractorFactory{})
}

/*
使用示例:

1. 创建Vcpkg提取器:
extractor := NewVcpkgExtractor("vcpkg.json")

2. 提取依赖:
deps, err := extractor.Extract()
if err != nil {
    log.Printf("Failed to extract dependencies: %v\n", err)
    return
}

3. 处理依赖信息:
for _, dep := range deps {
    fmt.Printf("Found dependency: %s %s (%s)\n", dep.Name, dep.Version, dep.Type)
    if len(dep.BuildFlags) > 0 {
        fmt.Printf("  Features: %v\n", dep.BuildFlags)
    }
}

示例vcpkg.json文件:
```json
{
    "name": "my-project",
    "version-string": "1.0.0",
    "dependencies": [
        {
            "name": "boost",
            "version-string": "1.76.0",
            "features": ["system", "filesystem"],
            "default-features": true
        },
        {
            "name": "openssl",
            "version-semver": "1.1.1",
            "platform": "windows"
        },
        {
            "name": "zlib",
            "version-date": "2021-05-25"
        }
    ],
    "features": {
        "test": {
            "description": "Build tests",
            "dependencies": [
                {
                    "name": "gtest",
                    "version-string": "1.10.0"
                }
            ]
        }
    },
    "default-features": ["test"],
    "overrides": [
        {
            "name": "boost",
            "version-string": "1.77.0"
        }
    ]
}
```
*/ 