package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSPMExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的Package.swift文件
	packageSwift := `{
		"name": "TestPackage",
		"platforms": [
			{
				"name": "ios",
				"version": "13.0"
			},
			{
				"name": "macos",
				"version": "10.15"
			}
		],
		"products": [
			{
				"name": "TestLib",
				"type": "library",
				"targets": ["TestLib"]
			}
		],
		"dependencies": [
			{
				"name": "Alamofire",
				"url": "https://github.com/Alamofire/Alamofire.git",
				"version": {
					"exact": "5.6.4"
				}
			},
			{
				"name": "SwiftyJSON",
				"url": "https://github.com/SwiftyJSON/SwiftyJSON.git",
				"version": {
					"lowerBound": "5.0.0",
					"upperBound": "6.0.0"
				}
			},
			{
				"name": "Kingfisher",
				"url": "https://github.com/onevcat/Kingfisher.git",
				"version": {
					"branch": "master"
				}
			}
		],
		"targets": [
			{
				"name": "TestLib",
				"type": "library",
				"dependencies": ["Alamofire", "SwiftyJSON", "Kingfisher"]
			},
			{
				"name": "TestLibTests",
				"type": "test",
				"dependencies": ["TestLib", "Quick", "Nimble"],
				"isTest": true
			}
		]
	}`

	err := os.WriteFile(filepath.Join(tempDir, "Package.swift"), []byte(packageSwift), 0644)
	require.NoError(t, err)

	// 创建测试用的Package.resolved文件
	packageResolved := `{
		"version": 1,
		"object": {
			"pins": [
				{
					"identity": "Alamofire",
					"location": "https://github.com/Alamofire/Alamofire.git",
					"state": {
						"version": "5.6.4",
						"checkedOut": true
					}
				},
				{
					"identity": "SwiftyJSON",
					"location": "https://github.com/SwiftyJSON/SwiftyJSON.git",
					"state": {
						"version": "5.0.1",
						"checkedOut": true
					}
				},
				{
					"identity": "Kingfisher",
					"location": "https://github.com/onevcat/Kingfisher.git",
					"state": {
						"branch": "master",
						"revision": "abc123def456",
						"checkedOut": true
					}
				},
				{
					"identity": "Quick",
					"location": "https://github.com/Quick/Quick.git",
					"state": {
						"version": "6.1.0",
						"checkedOut": true
					}
				},
				{
					"identity": "Nimble",
					"location": "https://github.com/Quick/Nimble.git",
					"state": {
						"version": "11.2.1",
						"checkedOut": true
					}
				}
			]
		}
	}`

	err = os.WriteFile(filepath.Join(tempDir, "Package.resolved"), []byte(packageResolved), 0644)
	require.NoError(t, err)

	// 创建SPM提取器实例
	extractor := NewSPMExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	require.NoError(t, err)

	// 验证提取的依赖信息
	assert.Len(t, deps, 3) // 应该有3个主要依赖

	// 验证Alamofire依赖
	alamofire := findDependency(deps, "Alamofire")
	require.NotNil(t, alamofire)
	assert.Equal(t, "spm", alamofire.Type)
	assert.Equal(t, "https://github.com/Alamofire/Alamofire.git", alamofire.Source)
	assert.Equal(t, "5.6.4", alamofire.Version)
	assert.Empty(t, alamofire.Scope)
	assert.Contains(t, alamofire.Metadata["platforms"], "ios 13.0")
	assert.Contains(t, alamofire.Metadata["platforms"], "macos 10.15")

	// 验证SwiftyJSON依赖
	swiftyJSON := findDependency(deps, "SwiftyJSON")
	require.NotNil(t, swiftyJSON)
	assert.Equal(t, "spm", swiftyJSON.Type)
	assert.Equal(t, "https://github.com/SwiftyJSON/SwiftyJSON.git", swiftyJSON.Source)
	assert.Equal(t, "5.0.1", swiftyJSON.Version) // 从Package.resolved获取的精确版本
	assert.Empty(t, swiftyJSON.Scope)

	// 验证Kingfisher依赖
	kingfisher := findDependency(deps, "Kingfisher")
	require.NotNil(t, kingfisher)
	assert.Equal(t, "spm", kingfisher.Type)
	assert.Equal(t, "https://github.com/onevcat/Kingfisher.git", kingfisher.Source)
	assert.Equal(t, "branch=master", kingfisher.Version)
	assert.Empty(t, kingfisher.Scope)
}

func TestSPMExtractor_ExtractWithoutPackageResolved(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的Package.swift文件
	packageSwift := `{
		"name": "TestPackage",
		"dependencies": [
			{
				"name": "Alamofire",
				"url": "https://github.com/Alamofire/Alamofire.git",
				"version": {
					"lowerBound": "5.0.0"
				}
			}
		]
	}`

	err := os.WriteFile(filepath.Join(tempDir, "Package.swift"), []byte(packageSwift), 0644)
	require.NoError(t, err)

	// 创建SPM提取器实例
	extractor := NewSPMExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	require.NoError(t, err)

	// 验证提取的依赖信息
	assert.Len(t, deps, 1)

	// 验证Alamofire依赖
	alamofire := findDependency(deps, "Alamofire")
	require.NotNil(t, alamofire)
	assert.Equal(t, "spm", alamofire.Type)
	assert.Equal(t, "https://github.com/Alamofire/Alamofire.git", alamofire.Source)
	assert.Equal(t, ">=5.0.0", alamofire.Version)
}

func TestSPMExtractor_ExtractWithTestDependencies(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的Package.swift文件
	packageSwift := `{
		"name": "TestPackage",
		"dependencies": [
			{
				"name": "Quick",
				"url": "https://github.com/Quick/Quick.git",
				"version": {
					"exact": "6.1.0"
				}
			},
			{
				"name": "Nimble",
				"url": "https://github.com/Quick/Nimble.git",
				"version": {
					"exact": "11.2.1"
				}
			}
		],
		"targets": [
			{
				"name": "TestLibTests",
				"type": "test",
				"dependencies": ["Quick", "Nimble"],
				"isTest": true
			}
		]
	}`

	err := os.WriteFile(filepath.Join(tempDir, "Package.swift"), []byte(packageSwift), 0644)
	require.NoError(t, err)

	// 创建SPM提取器实例
	extractor := NewSPMExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	require.NoError(t, err)

	// 验证提取的依赖信息
	assert.Len(t, deps, 2)

	// 验证Quick依赖
	quick := findDependency(deps, "Quick")
	require.NotNil(t, quick)
	assert.Equal(t, "dev", quick.Scope)
	assert.Equal(t, "6.1.0", quick.Version)

	// 验证Nimble依赖
	nimble := findDependency(deps, "Nimble")
	require.NotNil(t, nimble)
	assert.Equal(t, "dev", nimble.Scope)
	assert.Equal(t, "11.2.1", nimble.Version)
}

func TestSPMExtractor_ExtractInvalidPackageSwift(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建无效的Package.swift文件
	err := os.WriteFile(filepath.Join(tempDir, "Package.swift"), []byte("invalid json"), 0644)
	require.NoError(t, err)

	// 创建SPM提取器实例
	extractor := NewSPMExtractor()

	// 执行依赖提取
	_, err = extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse Package.swift")
}

func TestSPMExtractor_ExtractMissingPackageSwift(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建SPM提取器实例
	extractor := NewSPMExtractor()

	// 执行依赖提取
	_, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Package.swift not found")
}

// findDependency 辅助函数,用于在依赖列表中查找指定名称的依赖
func findDependency(deps []*models.Dependency, name string) *models.Dependency {
	for _, dep := range deps {
		if dep.Name == name {
			return dep
		}
	}
	return nil
}

// 注意事项:
// 1. 测试用例覆盖了主要功能和边缘情况
// 2. 使用临时目录避免污染文件系统
// 3. 测试数据尽量真实,模拟实际的Package.swift和Package.resolved
// 4. 验证所有重要的依赖属性
// 5. 包含错误处理的测试用例 