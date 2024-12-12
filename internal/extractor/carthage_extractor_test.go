package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/your-org/ccscanner/pkg/models"
)

func TestCarthageExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的Cartfile
	cartfile := `github "Alamofire/Alamofire" ~> 5.0
github "ReactiveX/RxSwift" == 6.5.0
github "Quick/Quick" "v5.0.1"
binary "https://my.domain.com/release/MyFramework.json" == 1.0
git "https://example.com/user/framework" "dev"
github "realm/realm-swift" "v10.32.3"`

	err := os.WriteFile(filepath.Join(tempDir, "Cartfile"), []byte(cartfile), 0644)
	assert.NoError(t, err)

	// 创建测试用的Cartfile.resolved文件
	cartfileResolved := `{
		"dependencies": [
			{
				"name": "Alamofire",
				"source": "git@github.com:Alamofire/Alamofire.git",
				"version": "5.6.4",
				"checksum": "d120af1e8638c7da36c8481fd61a66c2c08e3933"
			},
			{
				"name": "RxSwift",
				"source": "git@github.com:ReactiveX/RxSwift.git",
				"version": "6.5.0",
				"checksum": "b3dcd7dbd0d488e1a7077cb33b00f2083e382f07"
			},
			{
				"name": "Quick",
				"source": "git@github.com:Quick/Quick.git",
				"version": "v5.0.1",
				"checksum": "1234567890abcdef1234567890abcdef12345678"
			},
			{
				"name": "MyFramework",
				"source": "binary https://my.domain.com/release/MyFramework.json",
				"version": "1.0.0",
				"checksum": "abcdef1234567890abcdef1234567890abcdef12"
			},
			{
				"name": "framework",
				"source": "git@example.com:user/framework.git",
				"version": "abc123def456789abc123def456789abc123def4",
				"checksum": "f5e5f5c5d5b5a5c5d5e5f5c5d5b5a5c5d5e5f5c5"
			},
			{
				"name": "realm-swift",
				"source": "git@github.com:realm/realm-swift.git",
				"version": "v10.32.3",
				"checksum": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
			}
		]
	}`

	err = os.WriteFile(filepath.Join(tempDir, "Cartfile.resolved"), []byte(cartfileResolved), 0644)
	assert.NoError(t, err)

	// 创建Carthage/Build目录结构
	buildPath := filepath.Join(tempDir, "Carthage", "Build")
	err = os.MkdirAll(filepath.Join(buildPath, "iOS"), 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(filepath.Join(buildPath, "macOS"), 0755)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCarthageExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证提取的依赖
	expectedDeps := []*models.Dependency{
		{
			Name:    "Alamofire",
			Version: "5.6.4",
			Type:    "carthage",
			Source:  "git@github.com:Alamofire/Alamofire.git",
			Metadata: map[string]interface{}{
				"checksum":     "d120af1e8638c7da36c8481fd61a66c2c08e3933",
				"requirement": "~> 5.0",
				"platforms":    []string{"iOS", "macOS"},
			},
		},
		{
			Name:    "RxSwift",
			Version: "6.5.0",
			Type:    "carthage",
			Source:  "git@github.com:ReactiveX/RxSwift.git",
			Metadata: map[string]interface{}{
				"checksum":     "b3dcd7dbd0d488e1a7077cb33b00f2083e382f07",
				"requirement": "== 6.5.0",
				"platforms":    []string{"iOS", "macOS"},
			},
		},
		{
			Name:    "Quick",
			Version: "5.0.1", // 移除了前缀'v'
			Type:    "carthage",
			Source:  "git@github.com:Quick/Quick.git",
			Scope:   "dev", // 测试框架被标记为开发依赖
			Metadata: map[string]interface{}{
				"checksum":     "1234567890abcdef1234567890abcdef12345678",
				"requirement": "v5.0.1",
				"platforms":    []string{"iOS", "macOS"},
			},
		},
		{
			Name:    "MyFramework",
			Version: "1.0.0",
			Type:    "carthage",
			Source:  "binary=https://my.domain.com/release/MyFramework.json",
			Metadata: map[string]interface{}{
				"checksum":     "abcdef1234567890abcdef1234567890abcdef12",
				"requirement": "== 1.0",
				"platforms":    []string{"iOS", "macOS"},
			},
		},
		{
			Name:    "framework",
			Version: "commit=abc123def456789abc123def456789abc123def4",
			Type:    "carthage",
			Source:  "git@example.com:user/framework.git",
			Metadata: map[string]interface{}{
				"checksum":     "f5e5f5c5d5b5a5c5d5e5f5c5d5b5a5c5d5e5f5c5",
				"requirement": "dev",
				"platforms":    []string{"iOS", "macOS"},
			},
		},
		{
			Name:    "realm-swift",
			Version: "10.32.3", // 移除了前缀'v'
			Type:    "carthage",
			Source:  "git@github.com:realm/realm-swift.git",
			Metadata: map[string]interface{}{
				"checksum":     "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				"requirement": "v10.32.3",
				"platforms":    []string{"iOS", "macOS"},
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

func TestCarthageExtractor_ExtractNoCartfile(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建提取器实例
	extractor := NewCarthageExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "Cartfile not found")
}

func TestCarthageExtractor_ExtractNoCartfileResolved(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建空的Cartfile
	err := os.WriteFile(filepath.Join(tempDir, "Cartfile"), []byte(""), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCarthageExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "Cartfile.resolved not found")
}

func TestCarthageExtractor_ExtractInvalidCartfileResolved(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建空的Cartfile
	err := os.WriteFile(filepath.Join(tempDir, "Cartfile"), []byte(""), 0644)
	assert.NoError(t, err)

	// 创建无效的Cartfile.resolved
	invalidCartfileResolved := `{
		"dependencies": [
			{
				"name": "InvalidDep",
			}, // 无效的JSON
		]
	}`

	err = os.WriteFile(filepath.Join(tempDir, "Cartfile.resolved"), []byte(invalidCartfileResolved), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCarthageExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to parse Cartfile.resolved")
}

func TestCarthageExtractor_ExtractWithoutBuildDirectory(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建基本的Cartfile
	cartfile := `github "Alamofire/Alamofire" ~> 5.0`
	err := os.WriteFile(filepath.Join(tempDir, "Cartfile"), []byte(cartfile), 0644)
	assert.NoError(t, err)

	// 创建基本的Cartfile.resolved
	cartfileResolved := `{
		"dependencies": [
			{
				"name": "Alamofire",
				"source": "git@github.com:Alamofire/Alamofire.git",
				"version": "5.6.4"
			}
		]
	}`
	err = os.WriteFile(filepath.Join(tempDir, "Cartfile.resolved"), []byte(cartfileResolved), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCarthageExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证依赖列表
	assert.Equal(t, 1, len(deps))
	assert.Equal(t, "Alamofire", deps[0].Name)
	assert.Equal(t, "5.6.4", deps[0].Version)
	assert.Equal(t, "carthage", deps[0].Type)
	assert.Equal(t, "git@github.com:Alamofire/Alamofire.git", deps[0].Source)
	assert.NotContains(t, deps[0].Metadata, "platforms") // 没有平台信息
}