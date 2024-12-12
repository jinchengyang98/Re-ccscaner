package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuckExtractor_IsApplicable(t *testing.T) {
	extractor := NewBuckExtractor()
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "BUCK file",
			filePath: "path/to/BUCK",
			want:     true,
		},
		{
			name:     "BUCK.build file",
			filePath: "path/to/BUCK.build",
			want:     true,
		},
		{
			name:     "TARGETS file",
			filePath: "path/to/TARGETS",
			want:     true,
		},
		{
			name:     "Custom buck file",
			filePath: "path/to/custom.buck",
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

func TestBuckExtractor_Extract(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "BUCK")
	content := `
cpp_library(
    name = "my_lib",
    deps = [
        "//third-party/boost:boost",
        "//third-party/gtest:gtest",
    ],
)

cpp_binary(
    name = "my_binary",
    deps = [
        ":my_lib",
        "//third-party/protobuf:protobuf",
    ],
)

prebuilt_jar(
    name = "junit",
    binary_jar = "junit-4.12.jar",
    deps = [
        "//third-party/hamcrest:hamcrest",
    ],
)

remote_file(
    name = "boost_download",
    url = "https://boostorg.jfrog.io/artifactory/main/release/1.76.0/source/boost_1_76_0.tar.gz",
)
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewBuckExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]struct{}{
		"//third-party/boost:boost":       {},
		"//third-party/gtest:gtest":       {},
		":my_lib":                         {},
		"//third-party/protobuf:protobuf": {},
		"//third-party/hamcrest:hamcrest": {},
	}

	depsFound := make(map[string]bool)
	for _, dep := range deps {
		_, ok := expectedDeps[dep.Name]
		assert.True(t, ok, "Unexpected dependency: %s", dep.Name)
		assert.Equal(t, "buck_dependency", dep.Type)
		assert.Equal(t, testFile, dep.FilePath)
		depsFound[dep.Name] = true
	}

	// 验证所有预期的依赖都被找到
	for depName := range expectedDeps {
		assert.True(t, depsFound[depName], "Missing dependency: %s", depName)
	}
}

func TestBuckExtractor_Extract_WithComments(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "BUCK")
	content := `
cpp_library(
    name = "my_lib",
    deps = [
        # This is a comment
        "//third-party/boost:boost",  # Inline comment
        "//third-party/gtest:gtest",  // Another inline comment
    ],
)
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewBuckExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]struct{}{
		"//third-party/boost:boost": {},
		"//third-party/gtest:gtest": {},
	}

	assert.Equal(t, len(expectedDeps), len(deps))

	for _, dep := range deps {
		_, ok := expectedDeps[dep.Name]
		assert.True(t, ok, "Unexpected dependency: %s", dep.Name)
		assert.Equal(t, "buck_dependency", dep.Type)
		assert.Equal(t, testFile, dep.FilePath)
		assert.NotEmpty(t, dep.Parent)
	}
}

func TestBuckExtractor_Extract_EmptyFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建空文件
	testFile := filepath.Join(tempDir, "BUCK")
	err := os.WriteFile(testFile, []byte(""), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewBuckExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)
	assert.Empty(t, deps)
}

func TestBuckExtractor_Extract_InvalidFile(t *testing.T) {
	extractor := NewBuckExtractor()
	_, err := extractor.Extract("", "non_existent_file")
	assert.Error(t, err)
}

func TestBuckExtractor_GetName(t *testing.T) {
	extractor := NewBuckExtractor()
	assert.Equal(t, "Buck", extractor.GetName())
}

func TestBuckExtractor_GetPriority(t *testing.T) {
	extractor := NewBuckExtractor()
	assert.Equal(t, 100, extractor.GetPriority())
}

func TestBuckExtractor_String(t *testing.T) {
	extractor := NewBuckExtractor()
	assert.Contains(t, extractor.String(), "BuckExtractor")
	assert.Contains(t, extractor.String(), "Buck")
}

// 注意事项:
// 1. 测试用例覆盖了主要功能、边界情况和错误处理
// 2. 使用临时文件和目录进行测试
// 3. 测试了带注释的 Buck 文件解析
// 4. 验证了依赖关系的正确性
// 5. 检查了错误处理和边界条件

// 运行测试示例:
// go test -v ./internal/extractor/buck_extractor_test.go 