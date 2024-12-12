package extractor

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/ccscanner/pkg/models"
)

// NuGetExtractor 实现了.NET NuGet项目的依赖提取器
type NuGetExtractor struct {
	BaseExtractor
}

// ProjectFile 表示.NET项目文件的结构
type ProjectFile struct {
	XMLName     xml.Name `xml:"Project"`
	ItemGroups  []struct {
		PackageReferences []struct {
			Include string `xml:"Include,attr"`
			Version string `xml:"Version,attr"`
		} `xml:"PackageReference"`
	} `xml:"ItemGroup"`
}

// AssetsFile 表示project.assets.json的结构
type AssetsFile struct {
	Version   int `json:"version"`
	Targets   map[string]map[string]struct {
		Type     string `json:"type"`
		Dependencies map[string]string `json:"dependencies"`
		Runtime      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"runtime"`
	} `json:"targets"`
	Libraries map[string]struct {
		Type     string `json:"type"`
		Path     string `json:"path"`
		Files    []string `json:"files"`
		Sha512   string `json:"sha512"`
	} `json:"libraries"`
	Project struct {
		Version      string `json:"version"`
		Restore      struct {
			PackagesPath string `json:"packagesPath"`
		} `json:"restore"`
		Frameworks map[string]struct {
			Dependencies map[string]struct {
				Target  string `json:"target"`
				Version string `json:"version"`
			} `json:"dependencies"`
		} `json:"frameworks"`
	} `json:"project"`
}

// NewNuGetExtractor 创建一个新的NuGet提取器实例
func NewNuGetExtractor() *NuGetExtractor {
	return &NuGetExtractor{
		BaseExtractor: BaseExtractor{
			name: "nuget",
			patterns: []string{
				"*.csproj",
				"*.fsproj",
				"*.vbproj",
				"project.assets.json",
				"packages.config",
			},
		},
	}
}

// Extract 从项目文件和assets文件中提取依赖信息
func (e *NuGetExtractor) Extract(path string) ([]*models.Dependency, error) {
	var dependencies []*models.Dependency

	// 查找项目文件
	projectFiles, err := filepath.Glob(filepath.Join(path, "*.?sproj"))
	if err != nil {
		return nil, fmt.Errorf("failed to find project files: %v", err)
	}

	if len(projectFiles) == 0 {
		return nil, fmt.Errorf("no project files found")
	}

	// 处理每个项目文件
	for _, projectFile := range projectFiles {
		// 读取并解析项目文件
		projectData, err := os.ReadFile(projectFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read project file: %v", err)
		}

		var project ProjectFile
		if err := xml.Unmarshal(projectData, &project); err != nil {
			return nil, fmt.Errorf("failed to parse project file: %v", err)
		}

		// 提取PackageReference依赖
		for _, itemGroup := range project.ItemGroups {
			for _, packageRef := range itemGroup.PackageReferences {
				dependency := &models.Dependency{
					Name:    packageRef.Include,
					Version: packageRef.Version,
					Type:    "nuget",
				}
				dependencies = append(dependencies, dependency)
			}
		}
	}

	// 尝试从project.assets.json获取更详细的信息
	assetsPath := filepath.Join(path, "obj", "project.assets.json")
	if _, err := os.Stat(assetsPath); err == nil {
		assetsData, err := os.ReadFile(assetsPath)
		if err == nil {
			var assets AssetsFile
			if err := json.Unmarshal(assetsData, &assets); err == nil {
				// 更新依赖的详细信息
				for _, dep := range dependencies {
					// 在libraries中查找匹配的包
					for libKey, lib := range assets.Libraries {
						parts := strings.Split(libKey, "/")
						if len(parts) == 2 && parts[0] == dep.Name {
							dep.Version = parts[1]
							if lib.Path != "" {
								dep.Source = lib.Path
							}
							if lib.Sha512 != "" {
								if dep.Metadata == nil {
									dep.Metadata = make(map[string]interface{})
								}
								dep.Metadata["sha512"] = lib.Sha512
							}
							break
						}
					}

					// 在targets中查找依赖关系
					for _, target := range assets.Targets {
						for pkgKey, pkg := range target {
							parts := strings.Split(pkgKey, "/")
							if len(parts) == 2 && parts[0] == dep.Name {
								if len(pkg.Dependencies) > 0 {
									if dep.Metadata == nil {
										dep.Metadata = make(map[string]interface{})
									}
									dep.Metadata["dependencies"] = pkg.Dependencies
								}
								break
							}
						}
					}
				}
			}
		}
	}

	// 尝试从packages.config获取额外的依赖
	packagesConfigPath := filepath.Join(path, "packages.config")
	if _, err := os.Stat(packagesConfigPath); err == nil {
		packagesData, err := os.ReadFile(packagesConfigPath)
		if err == nil {
			var packagesConfig struct {
				XMLName  xml.Name `xml:"packages"`
				Packages []struct {
					ID                    string `xml:"id,attr"`
					Version              string `xml:"version,attr"`
					TargetFramework      string `xml:"targetFramework,attr"`
					DevelopmentDependency string `xml:"developmentDependency,attr"`
				} `xml:"package"`
			}
			if err := xml.Unmarshal(packagesData, &packagesConfig); err == nil {
				for _, pkg := range packagesConfig.Packages {
					// 检查是否已存在
					exists := false
					for _, dep := range dependencies {
						if dep.Name == pkg.ID {
							exists = true
							break
						}
					}

					if !exists {
						dependency := &models.Dependency{
							Name:    pkg.ID,
							Version: pkg.Version,
							Type:    "nuget",
						}

						if pkg.DevelopmentDependency == "true" {
							dependency.Scope = "dev"
						}

						if pkg.TargetFramework != "" {
							if dependency.Metadata == nil {
								dependency.Metadata = make(map[string]interface{})
							}
							dependency.Metadata["targetFramework"] = pkg.TargetFramework
						}

						dependencies = append(dependencies, dependency)
					}
				}
			}
		}
	}

	return dependencies, nil
} 