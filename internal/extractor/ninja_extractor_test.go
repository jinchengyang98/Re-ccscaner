package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNinjaExtractor_IsApplicable(t *testing.T) {
	extractor := NewNinjaExtractor()
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "build.ninja file",
			filePath: "path/to/build.ninja",
			want:     true,
		},
		{
			name:     "custom.ninja file",
			filePath: "path/to/custom.ninja",
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

func TestNinjaExtractor_Extract(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "build.ninja")
	content := `
# 变量定义
cxx = g++
cxxflags = -Wall -std=c++17
builddir = build

# 规则定义
rule cxx
  command = $cxx $cxxflags -c $in -o $out
  description = CXX $out

rule link
  command = $cxx $in -o $out
  description = LINK $out

# 构建语句
build $builddir/main.o: cxx src/main.cpp | src/config.h
build $builddir/utils.o: cxx src/utils.cpp | src/utils.h
build $builddir/app: link $builddir/main.o $builddir/utils.o | $builddir/lib.a

# 包含其他文件
include rules.ninja
subninja build/lib.ninja
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewNinjaExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]struct {
		Type   string
		Parent string
		Rule   string
	}{
		"src/main.cpp":        {Type: "ninja_input", Parent: "$builddir/main.o", Rule: "cxx"},
		"src/config.h":        {Type: "ninja_implicit", Parent: "$builddir/main.o", Rule: "cxx"},
		"src/utils.cpp":       {Type: "ninja_input", Parent: "$builddir/utils.o", Rule: "cxx"},
		"src/utils.h":         {Type: "ninja_implicit", Parent: "$builddir/utils.o", Rule: "cxx"},
		"$builddir/main.o":    {Type: "ninja_input", Parent: "$builddir/app", Rule: "link"},
		"$builddir/utils.o":   {Type: "ninja_input", Parent: "$builddir/app", Rule: "link"},
		"$builddir/lib.a":     {Type: "ninja_implicit", Parent: "$builddir/app", Rule: "link"},
		"rules.ninja":         {Type: "ninja_include", Parent: "", Rule: ""},
		"build/lib.ninja":     {Type: "ninja_subninja", Parent: "", Rule: ""},
	}

	assert.Equal(t, len(expectedDeps), len(deps))

	for _, dep := range deps {
		expected, ok := expectedDeps[dep.Name]
		assert.True(t, ok, "Unexpected dependency: %s", dep.Name)
		assert.Equal(t, expected.Type, dep.Type)
		assert.Equal(t, expected.Parent, dep.Parent)
		assert.Equal(t, expected.Rule, dep.Rule)
		assert.Equal(t, testFile, dep.FilePath)
	}
}

func TestNinjaExtractor_Extract_WithVariables(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "build.ninja")
	content := `
srcdir = src
objdir = build/obj
bindir = build/bin

build $objdir/main.o: cxx $srcdir/main.cpp
build ${bindir}/app: link ${objdir}/main.o
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewNinjaExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]struct {
		Type   string
		Parent string
	}{
		"src/main.cpp":          {Type: "ninja_input", Parent: "build/obj/main.o"},
		"build/obj/main.o":      {Type: "ninja_input", Parent: "build/bin/app"},
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

func TestNinjaExtractor_Extract_EmptyFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建空文件
	testFile := filepath.Join(tempDir, "build.ninja")
	err := os.WriteFile(testFile, []byte(""), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewNinjaExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)
	assert.Empty(t, deps)
}

func TestNinjaExtractor_Extract_InvalidFile(t *testing.T) {
	extractor := NewNinjaExtractor()
	_, err := extractor.Extract("", "non_existent_file")
	assert.Error(t, err)
}

func TestNinjaExtractor_GetName(t *testing.T) {
	extractor := NewNinjaExtractor()
	assert.Equal(t, "Ninja", extractor.GetName())
}

func TestNinjaExtractor_GetPriority(t *testing.T) {
	extractor := NewNinjaExtractor()
	assert.Equal(t, 100, extractor.GetPriority())
}

func TestNinjaExtractor_String(t *testing.T) {
	extractor := NewNinjaExtractor()
	assert.Contains(t, extractor.String(), "NinjaExtractor")
	assert.Contains(t, extractor.String(), "Ninja")
}

func TestNinjaExtractor_ExpandVariables(t *testing.T) {
	variables := map[string]string{
		"srcdir":  "src",
		"objdir":  "build/obj",
		"target":  "app",
		"CFLAGS": "-Wall -O2",
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Simple variable",
			input: "$srcdir/main.cpp",
			want:  "src/main.cpp",
		},
		{
			name:  "Braced variable",
			input: "${objdir}/main.o",
			want:  "build/obj/main.o",
		},
		{
			name:  "Multiple variables",
			input: "$objdir/${target}.o",
			want:  "build/obj/app.o",
		},
		{
			name:  "No variables",
			input: "main.cpp",
			want:  "main.cpp",
		},
		{
			name:  "Unknown variable",
			input: "$unknown/file",
			want:  "$unknown/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandVariables(tt.input, variables)
			assert.Equal(t, tt.want, got)
		})
	}
}

// 注意事项:
// 1. 测试覆盖了基本的 Ninja 构建文件语法
// 2. 测试了变量展开功能
// 3. 测试了显式和隐式依赖
// 4. 测试了 include 和 subninja 指令
// 5. 包含了错误处理和边界条件测试

// 运行测试示例:
// go test -v ./internal/extractor/ninja_extractor_test.go 