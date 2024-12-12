package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/your-org/ccscanner/pkg/models"
)

func TestCocoaPodsExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试用的Podfile
	podfile := `platform :ios, '13.0'

use_frameworks!

target 'MyApp' do
  pod 'Alamofire', '~> 5.0'
  pod 'SwiftyJSON', '~> 5.0'
  pod 'MyPrivatePod', :git => 'https://github.com/user/private-pod.git', :tag => 'v1.0.0'
  pod 'LocalPod', :path => '../LocalPod'

  target 'MyAppTests' do
    inherit! :search_paths
    pod 'Quick'
    pod 'Nimble'
  end
end`

	err := os.WriteFile(filepath.Join(tempDir, "Podfile"), []byte(podfile), 0644)
	assert.NoError(t, err)

	// 创建测试用的Podfile.lock文件
	podfileLock := `{
		"PODS": [
			{
				"name": "Alamofire (5.6.4)",
				"dependencies": [
					"Alamofire/Core (= 5.6.4)"
				]
			},
			{
				"name": "SwiftyJSON (5.0.1)"
			},
			{
				"name": "MyPrivatePod (1.0.0)"
			},
			{
				"name": "LocalPod (0.1.0)"
			},
			{
				"name": "Quick (5.0.1)",
				"dependencies": [
					"Nimble (~> 5.0.0)"
				]
			},
			{
				"name": "Nimble (5.0.0)"
			}
		],
		"DEPENDENCIES": [
			"Alamofire (~> 5.0)",
			"SwiftyJSON (~> 5.0)",
			"MyPrivatePod (from 'https://github.com/user/private-pod.git')",
			"LocalPod (from '../LocalPod')",
			"Quick",
			"Nimble"
		],
		"SPEC REPOS": {
			"trunk": [
				"Alamofire",
				"SwiftyJSON",
				"Quick",
				"Nimble"
			]
		},
		"EXTERNAL SOURCES": {
			"MyPrivatePod": {
				"git": "https://github.com/user/private-pod.git",
				"tag": "v1.0.0"
			},
			"LocalPod": {
				"path": "../LocalPod"
			}
		},
		"CHECKOUT OPTIONS": {
			"MyPrivatePod": {
				"git": "https://github.com/user/private-pod.git",
				"commit": "abc123def456"
			}
		},
		"SPEC CHECKSUMS": {
			"Alamofire": "d120af1e8638c7da36c8481fd61a66c2c08e3933",
			"SwiftyJSON": "b3dcd7dbd0d488e1a7077cb33b00f2083e382f07",
			"MyPrivatePod": "f5e5f5c5d5b5a5c5d5e5f5c5d5b5a5c5d5e5f5c5",
			"LocalPod": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			"Quick": "1234567890abcdef1234567890abcdef12345678",
			"Nimble": "abcdef1234567890abcdef1234567890abcdef12"
		},
		"PODFILE CHECKSUM": "0123456789abcdef0123456789abcdef01234567"
	}`

	err = os.WriteFile(filepath.Join(tempDir, "Podfile.lock"), []byte(podfileLock), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCocoaPodsExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, deps)

	// 验证提取的依赖
	expectedDeps := []*models.Dependency{
		{
			Name:    "Alamofire",
			Version: "5.6.4",
			Type:    "cocoapods",
			Source:  "trunk",
			Metadata: map[string]interface{}{
				"dependencies": []string{"Alamofire/Core (= 5.6.4)"},
				"checksum": "d120af1e8638c7da36c8481fd61a66c2c08e3933",
			},
		},
		{
			Name:    "SwiftyJSON",
			Version: "5.0.1",
			Type:    "cocoapods",
			Source:  "trunk",
			Metadata: map[string]interface{}{
				"checksum": "b3dcd7dbd0d488e1a7077cb33b00f2083e382f07",
			},
		},
		{
			Name:    "MyPrivatePod",
			Version: "tag=v1.0.0",
			Type:    "cocoapods",
			Source:  "https://github.com/user/private-pod.git",
			Metadata: map[string]interface{}{
				"checksum": "f5e5f5c5d5b5a5c5d5e5f5c5d5b5a5c5d5e5f5c5",
			},
		},
		{
			Name:    "LocalPod",
			Version: "0.1.0",
			Type:    "cocoapods",
			Source:  "path=../LocalPod",
			Metadata: map[string]interface{}{
				"checksum": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			},
		},
		{
			Name:    "Quick",
			Version: "5.0.1",
			Type:    "cocoapods",
			Source:  "trunk",
			Scope:   "dev",
			Metadata: map[string]interface{}{
				"dependencies": []string{"Nimble (~> 5.0.0)"},
				"checksum": "1234567890abcdef1234567890abcdef12345678",
			},
		},
		{
			Name:    "Nimble",
			Version: "5.0.0",
			Type:    "cocoapods",
			Source:  "trunk",
			Scope:   "dev",
			Metadata: map[string]interface{}{
				"checksum": "abcdef1234567890abcdef1234567890abcdef12",
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

func TestCocoaPodsExtractor_ExtractNoPodfile(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建提取器实例
	extractor := NewCocoaPodsExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "Podfile not found")
}

func TestCocoaPodsExtractor_ExtractNoPodfileLock(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建空的Podfile
	err := os.WriteFile(filepath.Join(tempDir, "Podfile"), []byte(""), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCocoaPodsExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "Podfile.lock not found")
}

func TestCocoaPodsExtractor_ExtractInvalidPodfileLock(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建空的Podfile
	err := os.WriteFile(filepath.Join(tempDir, "Podfile"), []byte(""), 0644)
	assert.NoError(t, err)

	// 创建无效的Podfile.lock
	invalidPodfileLock := `{
		"PODS": [
			{
				"name": "InvalidPod",
			}, // 无效的JSON
		]
	}`

	err = os.WriteFile(filepath.Join(tempDir, "Podfile.lock"), []byte(invalidPodfileLock), 0644)
	assert.NoError(t, err)

	// 创建提取器实例
	extractor := NewCocoaPodsExtractor()

	// 执行依赖提取
	deps, err := extractor.Extract(tempDir)
	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to parse Podfile.lock")
} 