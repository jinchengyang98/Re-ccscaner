package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/your-org/ccscanner/pkg/models"
)

func TestPoetryExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的pyproject.toml文件
	pyprojectToml := `{
		"tool": {
			"poetry": {
				"name": "test-project",
				"version": "0.1.0",
				"description": "A test project",
				"dependencies": {
					"requests": "^2.28.0",
					"fastapi": {
						"version": "^0.95.0",
						"extras": ["all"]
					},
					"custom-package": {
						"git": "https://github.com/user/repo",
						"branch": "main"
					}
				},
				"dev-dependencies": {
					"pytest": "^7.0.0",
					"black": {
						"version": "^23.0.0",
						"extras": ["d"]
					},
					"test-package": {
						"git": "https://github.com/user/test-repo",
						"tag": "v1.0.0"
					}
				}
			}
		}
	}`

	err := os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(pyprojectToml), 0644)
	assert.NoError(t, err)

	// 创建测试用的poetry.lock文件
	poetryLock := `{
		"package": [
			{
				"name": "requests",
				"version": "2.28.2",
				"category": "main",
				"source": {
					"type": "pypi",
					"url": "https://pypi.org/simple"
				}
			},
			{
				"name": "fastapi",
				"version": "0.95.1",
				"category": "main",
				"source": {
					"type": "pypi",
					"url": "https://pypi.org/simple"
				},
				"extras": ["all"]
			},
			{
				"name": "pytest",
				"version": "7.3.1",
				"category": "dev",
				"source": {
					"type": "pypi",
					"url": "https://pypi.org/simple"
				}
			},
			{
				"name": "black",
				"version": "23.3.0",
				"category": "dev",
				"source": {
					"type": "pypi",
					"url": "https://pypi.org/simple"
				},
				"extras": ["d"]
			}
		]
	}`

	err = os.WriteFile(filepath.Join(tempDir, "poetry.lock"), []byte(poetryLock), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewPoetryExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证提取的依赖
	expectedDeps := []*models.Dependency{
		{
			Name:    "requests",
			Version: "2.28.2",
			Type:    "poetry",
			Source:  "pypi+https://pypi.org/simple",
		},
		{
			Name:    "fastapi",
			Version: "0.95.1",
			Type:    "poetry",
			Source:  "pypi+https://pypi.org/simple",
			Metadata: map[string]interface{}{
				"extras": []string{"all"},
			},
		},
		{
			Name:    "custom-package",
			Version: "branch=main",
			Type:    "poetry",
			Source:  "https://github.com/user/repo",
		},
		{
			Name:    "pytest",
			Version: "7.3.1",
			Type:    "poetry",
			Source:  "pypi+https://pypi.org/simple",
			Scope:   "dev",
		},
		{
			Name:    "black",
			Version: "23.3.0",
			Type:    "poetry",
			Source:  "pypi+https://pypi.org/simple",
			Scope:   "dev",
			Metadata: map[string]interface{}{
				"extras": []string{"d"},
			},
		},
		{
			Name:    "test-package",
			Version: "tag=v1.0.0",
			Type:    "poetry",
			Source:  "https://github.com/user/test-repo",
			Scope:   "dev",
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

func TestPoetryExtractor_ExtractNoPyprojectToml(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建提取器实例
	extractor := NewPoetryExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "pyproject.toml not found")
}

func TestPoetryExtractor_ExtractInvalidPyprojectToml(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建无效的pyproject.toml文件
	invalidPyprojectToml := `{
		"tool": {
			"poetry": {
				"name": "test-project",
				"version": "0.1.0",
			}, // 无效的JSON
		}
	}`

	err := os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(invalidPyprojectToml), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewPoetryExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to parse pyproject.toml")
}

func TestPoetryExtractor_ExtractUnsupportedDependencyFormat(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建包含不支持的依赖格式的pyproject.toml文件
	pyprojectToml := `{
		"tool": {
			"poetry": {
				"name": "test-project",
				"version": "0.1.0",
				"dependencies": {
					"invalid-package": ["not", "supported"]
				}
			}
		}
	}`

	err := os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(pyprojectToml), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewPoetryExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "unsupported dependency format")
} 