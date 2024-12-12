package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/your-org/ccscanner/pkg/models"
)

func TestNuGetExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的.csproj文件
	csprojContent := `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.AspNetCore.App" Version="6.0.0" />
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
  <ItemGroup>
    <PackageReference Include="xunit" Version="2.4.1" />
    <PackageReference Include="Moq" Version="4.16.1" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(filepath.Join(tempDir, "test.csproj"), []byte(csprojContent), 0644)
	assert.NoError(t, err)

	// 创建obj目录
	err = os.MkdirAll(filepath.Join(tempDir, "obj"), 0755)
	assert.NoError(t, err)

	// 创建测试用的project.assets.json文件
	assetsContent := `{
		"version": 3,
		"targets": {
			".NETCoreApp,Version=v6.0": {
				"Microsoft.AspNetCore.App/6.0.0": {
					"type": "package",
					"dependencies": {
						"Microsoft.Extensions.Configuration": "6.0.0"
					}
				},
				"Newtonsoft.Json/13.0.1": {
					"type": "package"
				},
				"xunit/2.4.1": {
					"type": "package",
					"dependencies": {
						"xunit.core": "2.4.1"
					}
				},
				"Moq/4.16.1": {
					"type": "package"
				}
			}
		},
		"libraries": {
			"Microsoft.AspNetCore.App/6.0.0": {
				"type": "package",
				"path": "microsoft.aspnetcore.app/6.0.0",
				"files": ["lib/net6.0/Microsoft.AspNetCore.App.dll"],
				"sha512": "abc123"
			},
			"Newtonsoft.Json/13.0.1": {
				"type": "package",
				"path": "newtonsoft.json/13.0.1",
				"files": ["lib/netstandard2.0/Newtonsoft.Json.dll"],
				"sha512": "def456"
			}
		}
	}`

	err = os.WriteFile(filepath.Join(tempDir, "obj", "project.assets.json"), []byte(assetsContent), 0644)
	assert.NoError(t, err)

	// 创建测试用的packages.config文件
	packagesConfigContent := `<?xml version="1.0" encoding="utf-8"?>
<packages>
  <package id="NLog" version="4.7.13" targetFramework="net48" />
  <package id="StyleCop.Analyzers" version="1.1.118" targetFramework="net48" developmentDependency="true" />
</packages>`

	err = os.WriteFile(filepath.Join(tempDir, "packages.config"), []byte(packagesConfigContent), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewNuGetExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证提取的依赖
	expectedDeps := []*models.Dependency{
		{
			Name:    "Microsoft.AspNetCore.App",
			Version: "6.0.0",
			Type:    "nuget",
			Source:  "microsoft.aspnetcore.app/6.0.0",
			Metadata: map[string]interface{}{
				"sha512": "abc123",
				"dependencies": map[string]string{
					"Microsoft.Extensions.Configuration": "6.0.0",
				},
			},
		},
		{
			Name:    "Newtonsoft.Json",
			Version: "13.0.1",
			Type:    "nuget",
			Source:  "newtonsoft.json/13.0.1",
			Metadata: map[string]interface{}{
				"sha512": "def456",
			},
		},
		{
			Name:    "xunit",
			Version: "2.4.1",
			Type:    "nuget",
			Metadata: map[string]interface{}{
				"dependencies": map[string]string{
					"xunit.core": "2.4.1",
				},
			},
		},
		{
			Name:    "Moq",
			Version: "4.16.1",
			Type:    "nuget",
		},
		{
			Name:    "NLog",
			Version: "4.7.13",
			Type:    "nuget",
			Metadata: map[string]interface{}{
				"targetFramework": "net48",
			},
		},
		{
			Name:    "StyleCop.Analyzers",
			Version: "1.1.118",
			Type:    "nuget",
			Scope:   "dev",
			Metadata: map[string]interface{}{
				"targetFramework": "net48",
			},
		},
	}

	// 验证依赖列表
	assert.Equal(t, len(expectedDeps), len(deps))
	for i, dep := range deps {
		assert.Equal(t, expectedDeps[i].Name, dep.Name)
		assert.Equal(t, expectedDeps[i].Version, dep.Version)
		assert.Equal(t, expectedDeps[i].Type, dep.Type)
		assert.Equal(t, expectedDeps[i].Source, dep.Source)
		assert.Equal(t, expectedDeps[i].Scope, dep.Scope)
		assert.Equal(t, expectedDeps[i].Metadata, dep.Metadata)
	}
}

func TestNuGetExtractor_ExtractNoProjectFiles(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建提取器实例
	extractor := NewNuGetExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "no project files found")
}

func TestNuGetExtractor_ExtractInvalidProjectFile(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建无效的.csproj文件
	invalidCsproj := `<?xml version="1.0" encoding="utf-8"?>
<Project>
  <ItemGroup>
    <PackageReference Include="Invalid" Version="1.0.0" />
  </ItemGroup>
  <!-- 无效的XML -->
`

	err := os.WriteFile(filepath.Join(tempDir, "invalid.csproj"), []byte(invalidCsproj), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewNuGetExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to parse project file")
}

func TestNuGetExtractor_ExtractWithoutAssetsAndPackagesConfig(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建基本的.csproj文件
	csprojContent := `<?xml version="1.0" encoding="utf-8"?>
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`

	err := os.WriteFile(filepath.Join(tempDir, "basic.csproj"), []byte(csprojContent), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewNuGetExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证依赖列表
	assert.Equal(t, 1, len(deps))
	assert.Equal(t, "Newtonsoft.Json", deps[0].Name)
	assert.Equal(t, "13.0.1", deps[0].Version)
	assert.Equal(t, "nuget", deps[0].Type)
} 