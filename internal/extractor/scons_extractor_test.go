package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSconsExtractor_IsApplicable(t *testing.T) {
	extractor := NewSconsExtractor()
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "SConstruct file",
			filePath: "path/to/SConstruct",
			want:     true,
		},
		{
			name:     "SConscript file",
			filePath: "path/to/SConscript",
			want:     true,
		},
		{
			name:     "Custom scons file",
			filePath: "path/to/custom.scons",
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

func TestSconsExtractor_Extract(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "SConstruct")
	content := `
# 环境配置
env = Environment(
    LIBS = ['boost_system', 'boost_filesystem'],
    CPPPATH = ['include', '/usr/local/include'],
)

# 导入其他脚本
Import('custom_vars')
SConscript('src/SConscript')

# 构建目标
main_obj = env.Object('src/main.cpp')
utils_obj = env.Object('src/utils.cpp')

# 依赖声明
Depends(main_obj, 'src/config.h')
Requires(utils_obj, ['src/utils.h', 'src/common.h'])

# 库和程序
lib = env.Library('mylib', ['src/lib1.cpp', 'src/lib2.cpp'])
prog = env.Program('myapp', ['src/main.cpp', 'src/app.cpp'])

# 外部依赖
env.ParseConfig('pkg-config --cflags --libs openssl')
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewSconsExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]struct {
		Type   string
		Parent string
	}{
		"boost_system":        {Type: "scons_env", Parent: ""},
		"boost_filesystem":    {Type: "scons_env", Parent: ""},
		"custom_vars":         {Type: "scons_import", Parent: ""},
		"src/SConscript":      {Type: "scons_script", Parent: ""},
		"src/config.h":        {Type: "scons_depends", Parent: "src/main.cpp"},
		"src/utils.h":         {Type: "scons_requires", Parent: "src/utils.cpp"},
		"src/common.h":        {Type: "scons_requires", Parent: "src/utils.cpp"},
		"src/lib1.cpp":        {Type: "scons_library", Parent: "mylib"},
		"src/lib2.cpp":        {Type: "scons_library", Parent: "mylib"},
		"src/main.cpp":        {Type: "scons_program", Parent: "myapp"},
		"src/app.cpp":         {Type: "scons_program", Parent: "myapp"},
		"openssl":             {Type: "scons_pkg_config", Parent: ""},
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

func TestSconsExtractor_Extract_WithVariables(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "SConstruct")
	content := `
src_dir = 'src'
lib_dir = 'lib'
sources = ['main.cpp', 'app.cpp']

env = Environment()
env.Library(lib_dir + '/mylib', [src_dir + '/' + s for s in sources])
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewSconsExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)

	// 验证结果
	expectedDeps := map[string]struct {
		Type   string
		Parent string
	}{
		"src/main.cpp": {Type: "scons_library", Parent: "lib/mylib"},
		"src/app.cpp":  {Type: "scons_library", Parent: "lib/mylib"},
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

func TestSconsExtractor_Extract_EmptyFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建空文件
	testFile := filepath.Join(tempDir, "SConstruct")
	err := os.WriteFile(testFile, []byte(""), 0644)
	assert.NoError(t, err)

	// 运行测试
	extractor := NewSconsExtractor()
	deps, err := extractor.Extract(tempDir, testFile)
	assert.NoError(t, err)
	assert.Empty(t, deps)
}

func TestSconsExtractor_Extract_InvalidFile(t *testing.T) {
	extractor := NewSconsExtractor()
	_, err := extractor.Extract("", "non_existent_file")
	assert.Error(t, err)
}

func TestSconsExtractor_GetName(t *testing.T) {
	extractor := NewSconsExtractor()
	assert.Equal(t, "SCons", extractor.GetName())
}

func TestSconsExtractor_GetPriority(t *testing.T) {
	extractor := NewSconsExtractor()
	assert.Equal(t, 100, extractor.GetPriority())
}

func TestSconsExtractor_String(t *testing.T) {
	extractor := NewSconsExtractor()
	assert.Contains(t, extractor.String(), "SconsExtractor")
	assert.Contains(t, extractor.String(), "SCons")
}

func TestSconsExtractor_ExtractListItems(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "Simple list",
			input: "['a', 'b', 'c']",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Mixed quotes",
			input: `["a", 'b', "c"]`,
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "With spaces",
			input: "[ 'a' , 'b' , 'c' ]",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Single item",
			input: "['a']",
			want:  []string{"a"},
		},
		{
			name:  "Empty list",
			input: "[]",
			want:  nil,
		},
		{
			name:  "No brackets",
			input: "'a', 'b', 'c'",
			want:  []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractListItems(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSconsExtractor_ExtractPkgConfigPackages(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "Single package",
			command: "pkg-config --cflags --libs openssl",
			want:    []string{"openssl"},
		},
		{
			name:    "Multiple packages",
			command: "pkg-config --cflags --libs openssl libxml-2.0",
			want:    []string{"openssl", "libxml-2.0"},
		},
		{
			name:    "With version",
			command: "pkg-config --cflags --libs 'openssl >= 1.1.1'",
			want:    []string{"openssl >= 1.1.1"},
		},
		{
			name:    "Not pkg-config",
			command: "gcc -c main.cpp",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPkgConfigPackages(tt.command)
			assert.Equal(t, tt.want, got)
		})
	}
}

// 注意事项:
// 1. 测试覆盖了基本的 SCons 构建文件语法
// 2. 测试了变量展开功能
// 3. 测试了不同类型的依赖声明
// 4. 测试了列表解析功能
// 5. 包含了错误处理和边界条件测试

// 运行测试示例:
// go test -v ./internal/extractor/scons_extractor_test.go 