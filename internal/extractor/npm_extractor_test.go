package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNPMExtractor_Extract(t *testing.T) {
	// 创建临时测试目录
	testDir, err := os.MkdirTemp("", "npm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// 创建测试文件
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"dependencies": {
			"express": "^4.17.1",
			"lodash": "~4.17.21",
			"axios": "github:axios/axios#v0.21.1"
		},
		"devDependencies": {
			"jest": "^27.0.6",
			"typescript": "4.3.5"
		},
		"peerDependencies": {
			"react": ">=16.8.0",
			"react-dom": ">=16.8.0"
		},
		"optionalDependencies": {
			"fsevents": "^2.3.2"
		},
		"workspaces": [
			"packages/*"
		]
	}`

	packageLockJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"lockfileVersion": 2,
		"dependencies": {
			"express": {
				"version": "4.17.1",
				"resolved": "https://registry.npmjs.org/express/-/express-4.17.1.tgz",
				"integrity": "sha512-mHJ9O79RqluphRrcw2X/GTh3k9tVv8YcoyY4Kkh4WDMUYKRZUq0h1o0w2rrrxBqM7VoeUVqgb27xlEMXTnYt4g==",
				"dependencies": {
					"body-parser": "1.19.0",
					"cookie": "0.4.0"
				}
			},
			"lodash": {
				"version": "4.17.21",
				"resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz",
				"integrity": "sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs17LhbZVGedAJv8XZ1tvj5FvSg=="
			}
		}
	}`

	subPackageJSON := `{
		"name": "@test/sub-package",
		"version": "1.0.0",
		"dependencies": {
			"moment": "^2.29.1"
		}
	}`

	// 写入测试文件
	err = os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(testDir, "package-lock.json"), []byte(packageLockJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(testDir, "packages", "sub-package"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(testDir, "packages", "sub-package", "package.json"), []byte(subPackageJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 创建提取器
	logger, _ := zap.NewDevelopment()
	extractor := NewNPMExtractor(logger)

	// 执行测试
	deps, err := extractor.Extract(testDir)
	if err != nil {
		t.Fatal(err)
	}

	// 验证结果
	assert.NoError(t, err)
	assert.NotEmpty(t, deps)

	// 验证生产依赖
	found := false
	for _, dep := range deps {
		if dep.Name == "express" {
			found = true
			assert.Equal(t, "4.17.1", dep.Version)
			assert.Equal(t, "production", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "npm", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Production dependency not found")

	// 验证开发依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "jest" {
			found = true
			assert.Equal(t, "27.0.6", dep.Version)
			assert.Equal(t, "development", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "npm", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Development dependency not found")

	// 验证对等依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "react" {
			found = true
			assert.Equal(t, "16.8.0", dep.Version)
			assert.Equal(t, "peer", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "npm", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Peer dependency not found")

	// 验证可选依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "fsevents" {
			found = true
			assert.Equal(t, "2.3.2", dep.Version)
			assert.Equal(t, "optional", dep.Type)
			assert.False(t, dep.Required)
			assert.Equal(t, "npm", dep.BuildSystem)
		}
	}
	assert.True(t, found, "Optional dependency not found")

	// 验证锁定依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "express" && dep.Type == "locked" {
			found = true
			assert.Equal(t, "4.17.1", dep.Version)
			assert.True(t, dep.Required)
			assert.Equal(t, "npm", dep.BuildSystem)
			assert.Contains(t, dep.Source, "package-lock.json")
			assert.Contains(t, dep.Source, "registry.npmjs.org")
			assert.Len(t, dep.Dependencies, 2) // body-parser and cookie
		}
	}
	assert.True(t, found, "Locked dependency not found")

	// 验证工作区依赖
	found = false
	for _, dep := range deps {
		if dep.Name == "moment" {
			found = true
			assert.Equal(t, "2.29.1", dep.Version)
			assert.Equal(t, "production", dep.Type)
			assert.True(t, dep.Required)
			assert.Equal(t, "npm", dep.BuildSystem)
			assert.Contains(t, dep.Source, "sub-package")
		}
	}
	assert.True(t, found, "Workspace dependency not found")
}

func TestNPMExtractor_CleanVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "Caret range",
			version: "^1.2.3",
			want:    "1.2.3",
		},
		{
			name:    "Tilde range",
			version: "~1.2.3",
			want:    "1.2.3",
		},
		{
			name:    "Greater than",
			version: ">=1.2.3",
			want:    "1.2.3",
		},
		{
			name:    "Git URL",
			version: "github:user/repo#v1.2.3",
			want:    "github:user/repo#v1.2.3",
		},
		{
			name:    "File path",
			version: "file:../local-pkg",
			want:    "file:../local-pkg",
		},
		{
			name:    "Latest tag",
			version: "latest",
			want:    "latest",
		},
		{
			name:    "HTTP URL",
			version: "https://github.com/user/repo/archive/v1.2.3.tar.gz",
			want:    "https://github.com/user/repo/archive/v1.2.3.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanVersion(tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

/*
测试说明:

1. 主要测试用例:
- TestNPMExtractor_Extract: 测试完整的依赖提取功能
- TestNPMExtractor_CleanVersion: 测试版本号清理功能

2. 测试覆盖:
- package.json解析
- package-lock.json解析
- 工作区支持
- 各种类型的依赖
- 版本号格式处理

3. 测试数据:
- 模拟真实的package.json
- 模拟真实的package-lock.json
- 包含工作区
- 包含各种依赖类型
- 包含各种版本格式

4. 验证内容:
- 依赖解析的正确性
- 依赖属性的完整性
- 错误处理
- 边界情况

5. 运行方式:
go test -v ./internal/extractor -run "TestNPMExtractor"
*/ 