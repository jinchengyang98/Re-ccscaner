package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/your-org/ccscanner/pkg/models"
)

func TestCargoExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的Cargo.toml文件
	cargoToml := `{
		"package": {
			"name": "test-project",
			"version": "0.1.0"
		},
		"dependencies": {
			"serde": {
				"version": "1.0.0",
				"features": ["derive"]
			},
			"tokio": {
				"version": "1.0",
				"features": ["full"]
			},
			"git-dep": {
				"git": "https://github.com/user/repo",
				"branch": "main"
			}
		},
		"dev-dependencies": {
			"mockall": {
				"version": "0.11.0"
			},
			"git-test-dep": {
				"git": "https://github.com/user/test-repo",
				"rev": "abc123"
			}
		}
	}`

	err := os.WriteFile(filepath.Join(tempDir, "Cargo.toml"), []byte(cargoToml), 0644)
	assert.NoError(t, err)

	// 创建测试用的Cargo.lock文件
	cargoLock := `{
		"package": [
			{
				"name": "serde",
				"version": "1.0.152",
				"source": "registry+https://github.com/rust-lang/crates.io-index"
			},
			{
				"name": "tokio",
				"version": "1.25.0",
				"source": "registry+https://github.com/rust-lang/crates.io-index"
			},
			{
				"name": "mockall",
				"version": "0.11.3",
				"source": "registry+https://github.com/rust-lang/crates.io-index"
			}
		]
	}`

	err = os.WriteFile(filepath.Join(tempDir, "Cargo.lock"), []byte(cargoLock), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCargoExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证提取的依赖
	expectedDeps := []*models.Dependency{
		{
			Name:    "serde",
			Version: "1.0.152",
			Type:    "cargo",
			Source:  "registry+https://github.com/rust-lang/crates.io-index",
			Metadata: map[string]interface{}{
				"features": []string{"derive"},
			},
		},
		{
			Name:    "tokio",
			Version: "1.25.0",
			Type:    "cargo",
			Source:  "registry+https://github.com/rust-lang/crates.io-index",
			Metadata: map[string]interface{}{
				"features": []string{"full"},
			},
		},
		{
			Name:    "git-dep",
			Version: "branch=main",
			Type:    "cargo",
			Source:  "https://github.com/user/repo",
		},
		{
			Name:    "mockall",
			Version: "0.11.3",
			Type:    "cargo",
			Source:  "registry+https://github.com/rust-lang/crates.io-index",
			Scope:   "dev",
		},
		{
			Name:    "git-test-dep",
			Version: "rev=abc123",
			Type:    "cargo",
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

func TestCargoExtractor_ExtractNoCargoToml(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建提取器实例
	extractor := NewCargoExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "Cargo.toml not found")
}

func TestCargoExtractor_ExtractInvalidCargoToml(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建无效的Cargo.toml文件
	invalidCargoToml := `{
		"package": {
			"name": "test-project",
			"version": "0.1.0",
		}, // 无效的JSON
	}`

	err := os.WriteFile(filepath.Join(tempDir, "Cargo.toml"), []byte(invalidCargoToml), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCargoExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to parse Cargo.toml")
} 