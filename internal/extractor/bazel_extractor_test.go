package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBazelExtractor_IsApplicable(t *testing.T) {
	extractor := NewBazelExtractor()
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "BUILD file",
			filePath: "path/to/BUILD",
			want:     true,
		},
		{
			name:     "BUILD.bazel file",
			filePath: "path/to/BUILD.bazel",
			want:     true,
		},
		{
			name:     "WORKSPACE file",
			filePath: "path/to/WORKSPACE",
			want:     true,
		},
		{
			name:     "WORKSPACE.bazel file",
			filePath: "path/to/WORKSPACE.bazel",
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

func TestBazelExtractor_Extract(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "WORKSPACE")
	content := `
workspace(name = "my_workspace")

http_archive(
    name = "com_google_googletest",
    urls = ["https://github.com/google/googletest/archive/release-1.10.0.zip"],
)

git_repository(
    name = "com_github_gflags_gflags",
    remote = "https://github.com/gflags/gflags.git",
    tag = "v2.2.2",
)

local_repository(
    name = "my_local_repo",
    path = "/path/to/local/repo",
)

maven_jar(
    name = "junit_junit",
    artifact = "junit:junit:4.12",
)
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewBazelExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]string{
		"com_google_googletest":      "bazel_http_archive",
		"com_github_gflags_gflags":   "bazel_git_repository",
		"my_local_repo":              "bazel_local_repository",
		"junit_junit":                "bazel_maven_jar",
	}

	assert.Equal(t, len(expectedDeps), len(deps))

	for _, dep := range deps {
		expectedType, ok := expectedDeps[dep.Name]
		assert.True(t, ok, "Unexpected dependency: %s", dep.Name)
		assert.Equal(t, expectedType, dep.Type)
		assert.Equal(t, testFile, dep.FilePath)
	}
}

func TestBazelExtractor_Extract_EmptyFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建空文件
	testFile := filepath.Join(tempDir, "WORKSPACE")
	err := os.WriteFile(testFile, []byte(""), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewBazelExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)
	assert.Empty(t, deps)
}

func TestBazelExtractor_Extract_InvalidFile(t *testing.T) {
	extractor := NewBazelExtractor()
	_, err := extractor.Extract("", "non_existent_file")
	assert.Error(t, err)
}

func TestBazelExtractor_GetName(t *testing.T) {
	extractor := NewBazelExtractor()
	assert.Equal(t, "Bazel", extractor.GetName())
}

func TestBazelExtractor_GetPriority(t *testing.T) {
	extractor := NewBazelExtractor()
	assert.Equal(t, 100, extractor.GetPriority())
}

func TestBazelExtractor_String(t *testing.T) {
	extractor := NewBazelExtractor()
	assert.Contains(t, extractor.String(), "BazelExtractor")
	assert.Contains(t, extractor.String(), "Bazel")
}

// 注意事项:
// 1. 测试用例应该覆盖所有主要功能和边界情况
// 2. 使用临时文件和目录来避免影响实际文件系统
// 3. 清理测试产生的临时文件和目录
// 4. 使用 assert 包来简化测试断言
// 5. 添加足够的注释来解释测试的目的和预期结果

// 运行测试示例:
// go test -v ./internal/extractor/bazel_extractor_test.go 