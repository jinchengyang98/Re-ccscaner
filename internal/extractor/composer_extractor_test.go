package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/your-org/ccscanner/pkg/models"
)

func TestComposerExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的composer.json文件
	composerJson := `{
		"name": "test/project",
		"description": "A test project",
		"type": "project",
		"require": {
			"php": ">=7.4",
			"laravel/framework": "^8.0",
			"guzzlehttp/guzzle": "^7.0",
			"ext-json": "*"
		},
		"require-dev": {
			"phpunit/phpunit": "^9.0",
			"mockery/mockery": "^1.4",
			"ext-xdebug": "*"
		},
		"repositories": [
			{
				"type": "composer",
				"url": "https://packagist.org"
			}
		]
	}`

	err := os.WriteFile(filepath.Join(tempDir, "composer.json"), []byte(composerJson), 0644)
	assert.NoError(t, err)

	// 创建测试用的composer.lock文件
	composerLock := `{
		"packages": [
			{
				"name": "laravel/framework",
				"version": "8.83.27",
				"source": {
					"type": "git",
					"url": "https://github.com/laravel/framework.git",
					"reference": "e1afe088b4ca613fb96dc57e6d8dbcb8c6ee4ff2"
				},
				"dist": {
					"type": "zip",
					"url": "https://api.github.com/repos/laravel/framework/zipball/e1afe088b4ca613fb96dc57e6d8dbcb8c6ee4ff2",
					"reference": "e1afe088b4ca613fb96dc57e6d8dbcb8c6ee4ff2",
					"shasum": ""
				},
				"require": {
					"php": "^7.3|^8.0",
					"ext-mbstring": "*"
				},
				"type": "library",
				"time": "2023-05-09T13:41:51+00:00"
			},
			{
				"name": "guzzlehttp/guzzle",
				"version": "7.5.1",
				"source": {
					"type": "git",
					"url": "https://github.com/guzzle/guzzle.git",
					"reference": "b964ca597e86b752cd994f27293e9fa6b6a95ed9"
				},
				"dist": {
					"type": "zip",
					"url": "https://api.github.com/repos/guzzle/guzzle/zipball/b964ca597e86b752cd994f27293e9fa6b6a95ed9",
					"reference": "b964ca597e86b752cd994f27293e9fa6b6a95ed9",
					"shasum": ""
				},
				"require": {
					"php": "^7.2.5 || ^8.0",
					"ext-json": "*"
				},
				"type": "library",
				"time": "2023-04-17T16:30:08+00:00"
			}
		],
		"packages-dev": [
			{
				"name": "phpunit/phpunit",
				"version": "9.6.7",
				"source": {
					"type": "git",
					"url": "https://github.com/sebastianbergmann/phpunit.git",
					"reference": "c993f0d3b0489ffc42ee2fe0bd645af1538a63b2"
				},
				"dist": {
					"type": "zip",
					"url": "https://api.github.com/repos/sebastianbergmann/phpunit/zipball/c993f0d3b0489ffc42ee2fe0bd645af1538a63b2",
					"reference": "c993f0d3b0489ffc42ee2fe0bd645af1538a63b2",
					"shasum": ""
				},
				"require": {
					"php": ">=7.3",
					"ext-dom": "*"
				},
				"type": "library",
				"time": "2023-04-27T07:28:15+00:00"
			},
			{
				"name": "mockery/mockery",
				"version": "1.5.1",
				"source": {
					"type": "git",
					"url": "https://github.com/mockery/mockery.git",
					"reference": "e92dcc83d5a51851baf5f5591d32cb2b16e3684e"
				},
				"dist": {
					"type": "zip",
					"url": "https://api.github.com/repos/mockery/mockery/zipball/e92dcc83d5a51851baf5f5591d32cb2b16e3684e",
					"reference": "e92dcc83d5a51851baf5f5591d32cb2b16e3684e",
					"shasum": ""
				},
				"require": {
					"php": "^7.3 || ^8.0"
				},
				"type": "library",
				"time": "2022-09-07T15:32:08+00:00"
			}
		]
	}`

	err = os.WriteFile(filepath.Join(tempDir, "composer.lock"), []byte(composerLock), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewComposerExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证提取的依赖
	expectedDeps := []*models.Dependency{
		{
			Name:    "laravel/framework",
			Version: "8.83.27",
			Type:    "composer",
			Source:  "git+https://github.com/laravel/framework.git",
			Metadata: map[string]interface{}{
				"require": map[string]string{
					"php":          "^7.3|^8.0",
					"ext-mbstring": "*",
				},
			},
		},
		{
			Name:    "guzzlehttp/guzzle",
			Version: "7.5.1",
			Type:    "composer",
			Source:  "git+https://github.com/guzzle/guzzle.git",
			Metadata: map[string]interface{}{
				"require": map[string]string{
					"php":       "^7.2.5 || ^8.0",
					"ext-json": "*",
				},
			},
		},
		{
			Name:    "phpunit/phpunit",
			Version: "9.6.7",
			Type:    "composer",
			Source:  "git+https://github.com/sebastianbergmann/phpunit.git",
			Scope:   "dev",
			Metadata: map[string]interface{}{
				"require": map[string]string{
					"php":      ">=7.3",
					"ext-dom": "*",
				},
			},
		},
		{
			Name:    "mockery/mockery",
			Version: "1.5.1",
			Type:    "composer",
			Source:  "git+https://github.com/mockery/mockery.git",
			Scope:   "dev",
			Metadata: map[string]interface{}{
				"require": map[string]string{
					"php": "^7.3 || ^8.0",
				},
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

func TestComposerExtractor_ExtractNoComposerJson(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建提取器实例
	extractor := NewComposerExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "composer.json not found")
}

func TestComposerExtractor_ExtractInvalidComposerJson(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建无效的composer.json文件
	invalidComposerJson := `{
		"name": "test/project",
		"require": {
			"php": ">=7.4",
		}, // 无效的JSON
	}`

	err := os.WriteFile(filepath.Join(tempDir, "composer.json"), []byte(invalidComposerJson), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewComposerExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to parse composer.json")
}

func TestComposerExtractor_ExtractWithoutLockFile(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建composer.json文件
	composerJson := `{
		"name": "test/project",
		"require": {
			"laravel/framework": "^8.0",
			"guzzlehttp/guzzle": "^7.0"
		},
		"require-dev": {
			"phpunit/phpunit": "^9.0"
		}
	}`

	err := os.WriteFile(filepath.Join(tempDir, "composer.json"), []byte(composerJson), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewComposerExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证依赖列表
	assert.Equal(t, 3, len(deps))
	assert.Equal(t, "^8.0", deps[0].Version) // 使用composer.json中的版本约束
} 