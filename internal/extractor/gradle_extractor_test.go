package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGradleExtractor_IsApplicable(t *testing.T) {
	extractor := NewGradleExtractor()
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "build.gradle file",
			filePath: "path/to/build.gradle",
			want:     true,
		},
		{
			name:     "build.gradle.kts file",
			filePath: "path/to/build.gradle.kts",
			want:     true,
		},
		{
			name:     "settings.gradle file",
			filePath: "path/to/settings.gradle",
			want:     true,
		},
		{
			name:     "settings.gradle.kts file",
			filePath: "path/to/settings.gradle.kts",
			want:     true,
		},
		{
			name:     "Other file",
			filePath: "path/to/other.txt",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.IsApplicable(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGradleExtractor_Extract_BuildGradle(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "build.gradle")
	content := `
plugins {
    id 'cpp-library'
    id 'cpp-unit-test'
    id 'com.example.plugin' version '1.2.3'
}

components.native {
    main {
        dependencies {
            nativeImplementation 'boost:boost:1.76.0'
            nativeLib 'openssl:openssl:1.1.1'
        }

        cppCompiler.includeDirs.from('include')
    }

    test {
        dependencies {
            nativeImplementation project(':core')
            nativeApi 'gtest:gtest:1.10.0'
        }
    }
}
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewGradleExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]struct {
		Type   string
		Parent string
	}{
		"boost:boost:1.76.0":           {Type: "gradle_native", Parent: "main"},
		"openssl:openssl:1.1.1":        {Type: "gradle_native", Parent: "main"},
		":core":                        {Type: "gradle_project", Parent: ""},
		"gtest:gtest:1.10.0":          {Type: "gradle_native", Parent: "test"},
		"include":                      {Type: "gradle_include_dir", Parent: "main"},
		"com.example.plugin":           {Type: "gradle_plugin", Parent: ""},
	}

	assert.Equal(t, len(expectedDeps), len(deps))

	for _, dep := range deps {
		expected, ok := expectedDeps[dep.Name]
		assert.True(t, ok, "Unexpected dependency: %s", dep.Name)
		assert.Equal(t, expected.Type, dep.Type)
		assert.Equal(t, expected.Parent, dep.Parent)
		assert.Equal(t, testFile, dep.FilePath)
	}
}

func TestGradleExtractor_Extract_SettingsGradle(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "settings.gradle")
	content := `
rootProject.name = 'my-project'

include ':app'
include ':core'
include ':lib:common'
include ':lib:network'
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewGradleExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedSubprojects := []string{
		":app",
		":core",
		":lib:common",
		":lib:network",
	}

	assert.Equal(t, len(expectedSubprojects), len(deps))

	for i, dep := range deps {
		assert.Equal(t, expectedSubprojects[i], dep.Name)
		assert.Equal(t, "gradle_subproject", dep.Type)
		assert.Equal(t, testFile, dep.FilePath)
	}
}

func TestGradleExtractor_Extract_KotlinDSL(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "build.gradle.kts")
	content := `
plugins {
    id("cpp-library")
    id("cpp-unit-test")
    id("com.example.plugin") version "1.2.3"
}

components.native {
    main {
        dependencies {
            nativeImplementation("boost:boost:1.76.0")
            nativeLib("openssl:openssl:1.1.1")
        }

        cppCompiler.includeDirs.from("include")
    }

    test {
        dependencies {
            nativeImplementation(project(":core"))
            nativeApi("gtest:gtest:1.10.0")
        }
    }
}
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewGradleExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]struct {
		Type   string
		Parent string
	}{
		"boost:boost:1.76.0":           {Type: "gradle_native", Parent: "main"},
		"openssl:openssl:1.1.1":        {Type: "gradle_native", Parent: "main"},
		":core":                        {Type: "gradle_project", Parent: ""},
		"gtest:gtest:1.10.0":          {Type: "gradle_native", Parent: "test"},
		"include":                      {Type: "gradle_include_dir", Parent: "main"},
		"com.example.plugin":           {Type: "gradle_plugin", Parent: ""},
	}

	assert.Equal(t, len(expectedDeps), len(deps))

	for _, dep := range deps {
		expected, ok := expectedDeps[dep.Name]
		assert.True(t, ok, "Unexpected dependency: %s", dep.Name)
		assert.Equal(t, expected.Type, dep.Type)
		assert.Equal(t, expected.Parent, dep.Parent)
		assert.Equal(t, testFile, dep.FilePath)
	}
}

func TestGradleExtractor_Extract_EmptyFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建空文件
	testFile := filepath.Join(tempDir, "build.gradle")
	err := os.WriteFile(testFile, []byte(""), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewGradleExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)
	assert.Empty(t, deps)
}

func TestGradleExtractor_Extract_InvalidFile(t *testing.T) {
	extractor := NewGradleExtractor()
	_, err := extractor.Extract("", "non_existent_file")
	assert.Error(t, err)
}

func TestGradleExtractor_GetName(t *testing.T) {
	extractor := NewGradleExtractor()
	assert.Equal(t, "Gradle", extractor.GetName())
}

func TestGradleExtractor_GetPriority(t *testing.T) {
	extractor := NewGradleExtractor()
	assert.Equal(t, 100, extractor.GetPriority())
}

func TestGradleExtractor_String(t *testing.T) {
	extractor := NewGradleExtractor()
	assert.Contains(t, extractor.String(), "GradleExtractor")
	assert.Contains(t, extractor.String(), "Gradle")
}

// 注意事项:
// 1. 测试覆盖了 Groovy 和 Kotlin DSL 两种语法
// 2. 测试了不同类型的依赖声明
// 3. 测试了多项目构建的情况
// 4. 包含了错误处理和边界条件测试
// 5. 使用临时文件和目录进行测试

// 运行测试示例:
// go test -v ./internal/extractor/gradle_extractor_test.go 