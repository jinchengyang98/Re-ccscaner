package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMultipleExtractors_Integration 测试多个提取器一起工作的情况
func TestMultipleExtractors_Integration(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试文件结构
	files := map[string]string{
		"build.gradle": `
			plugins {
				id 'cpp-library'
				id 'cpp-unit-test'
			}
			
			components.native {
				main {
					dependencies {
						nativeImplementation 'boost:boost:1.76.0'
						nativeLib 'openssl:openssl:1.1.1'
					}
				}
			}
		`,
		"build.ninja": `
			cxx = g++
			cxxflags = -Wall -std=c++17
			
			rule cxx
				command = $cxx $cxxflags -c $in -o $out
			
			build obj/main.o: cxx src/main.cpp | src/config.h
			build obj/utils.o: cxx src/utils.cpp | src/utils.h
		`,
		"SConstruct": `
			env = Environment(
				LIBS = ['boost_system', 'boost_filesystem'],
			)
			
			lib = env.Library('mylib', ['src/lib1.cpp', 'src/lib2.cpp'])
			prog = env.Program('myapp', ['src/main.cpp', 'src/app.cpp'])
		`,
	}

	// 写入测试文件
	for name, content := range files {
		filePath := filepath.Join(tempDir, name)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// 创建提取器实例
	extractors := []Extractor{
		NewGradleExtractor(),
		NewNinjaExtractor(),
		NewSconsExtractor(),
	}

	// 收集所有依赖
	var allDeps []models.Dependency
	for _, extractor := range extractors {
		for name := range files {
			filePath := filepath.Join(tempDir, name)
			if extractor.IsApplicable(filePath) {
				deps, err := extractor.Extract(tempDir, filePath)
				assert.NoError(t, err)
				allDeps = append(allDeps, deps...)
			}
		}
	}

	// 验证依赖数量和类型
	var (
		gradleDeps  int
		ninjaDeps   int
		sconsDeps   int
		totalDeps   = len(allDeps)
		uniqueDeps  = make(map[string]bool)
		uniqueTypes = make(map[string]bool)
	)

	for _, dep := range allDeps {
		uniqueDeps[dep.Name] = true
		uniqueTypes[dep.Type] = true

		switch {
		case dep.Type == "gradle_native" || dep.Type == "gradle_plugin":
			gradleDeps++
		case dep.Type == "ninja_input" || dep.Type == "ninja_implicit":
			ninjaDeps++
		case dep.Type == "scons_library" || dep.Type == "scons_program" || dep.Type == "scons_env":
			sconsDeps++
		}
	}

	// 验证每个提取器都找到了依赖
	assert.Greater(t, gradleDeps, 0, "Gradle extractor should find dependencies")
	assert.Greater(t, ninjaDeps, 0, "Ninja extractor should find dependencies")
	assert.Greater(t, sconsDeps, 0, "SCons extractor should find dependencies")

	// 验证依赖类型的多样性
	assert.Greater(t, len(uniqueTypes), 5, "Should find multiple dependency types")

	// 验证没有重复依赖
	assert.Equal(t, len(uniqueDeps), totalDeps, "Should not have duplicate dependencies")
}

// TestExtractors_CrossProjectDependencies 测试跨项目依赖的情况
func TestExtractors_CrossProjectDependencies(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建项目结构
	projectStructure := map[string]string{
		"core/build.gradle": `
			plugins {
				id 'cpp-library'
			}
			
			components.native {
				main {
					dependencies {
						nativeImplementation 'boost:boost:1.76.0'
					}
				}
			}
		`,
		"core/build.ninja": `
			build obj/core.o: cxx src/core.cpp | src/core.h
		`,
		"app/SConstruct": `
			env = Environment()
			env.Program('app', ['main.cpp', '../core/lib/libcore.a'])
		`,
		"app/build.gradle": `
			dependencies {
				implementation project(':core')
			}
		`,
	}

	// 写入项目文件
	for path, content := range projectStructure {
		filePath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// 创建提取器实例
	extractors := []Extractor{
		NewGradleExtractor(),
		NewNinjaExtractor(),
		NewSconsExtractor(),
	}

	// 收集所有依赖
	var allDeps []models.Dependency
	for _, extractor := range extractors {
		err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && extractor.IsApplicable(path) {
				deps, err := extractor.Extract(tempDir, path)
				assert.NoError(t, err)
				allDeps = append(allDeps, deps...)
			}
			return nil
		})
		assert.NoError(t, err)
	}

	// 验证跨项目依赖
	var (
		crossProjectDeps int
		externalDeps    int
	)

	for _, dep := range allDeps {
		if dep.Type == "gradle_project" || strings.Contains(dep.Name, "../") {
			crossProjectDeps++
		}
		if strings.Contains(dep.Name, "boost") || strings.Contains(dep.Name, "lib") {
			externalDeps++
		}
	}

	assert.Greater(t, crossProjectDeps, 0, "Should find cross-project dependencies")
	assert.Greater(t, externalDeps, 0, "Should find external dependencies")
}

// TestExtractors_Performance 测试提取器的性能
func TestExtractors_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// 创建临时目录
	tempDir := t.TempDir()

	// 生成大量测试文件
	numFiles := 100
	filesPerType := numFiles / 3

	// 生成 Gradle 文件
	for i := 0; i < filesPerType; i++ {
		content := generateLargeGradleFile(50) // 每个文件50个依赖
		filePath := filepath.Join(tempDir, fmt.Sprintf("module%d/build.gradle", i))
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// 生成 Ninja 文件
	for i := 0; i < filesPerType; i++ {
		content := generateLargeNinjaFile(50) // 每个文件50个构建规则
		filePath := filepath.Join(tempDir, fmt.Sprintf("module%d/build.ninja", i))
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// 生成 SCons 文件
	for i := 0; i < filesPerType; i++ {
		content := generateLargeSConsFile(50) // 每个文件50个目标
		filePath := filepath.Join(tempDir, fmt.Sprintf("module%d/SConstruct", i))
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// 创建提取器实例
	extractors := []Extractor{
		NewGradleExtractor(),
		NewNinjaExtractor(),
		NewSconsExtractor(),
	}

	// 测试每个提取器的性能
	for _, extractor := range extractors {
		start := time.Now()
		var totalDeps int

		err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && extractor.IsApplicable(path) {
				deps, err := extractor.Extract(tempDir, path)
				assert.NoError(t, err)
				totalDeps += len(deps)
			}
			return nil
		})
		assert.NoError(t, err)

		duration := time.Since(start)
		t.Logf("%s processed %d dependencies in %v (%.2f deps/sec)",
			extractor.GetName(),
			totalDeps,
			duration,
			float64(totalDeps)/duration.Seconds())

		// 验证性能指标
		assert.Less(t, duration, 5*time.Second, "%s should process files within 5 seconds", extractor.GetName())
		assert.Greater(t, float64(totalDeps)/duration.Seconds(), 1000.0,
			"%s should process at least 1000 dependencies per second", extractor.GetName())
	}
}

// 生成大型测试文件的辅助函数
func generateLargeGradleFile(numDeps int) string {
	var sb strings.Builder
	sb.WriteString("plugins { id 'cpp-library' }\n\ncomponents.native {\n    main {\n        dependencies {\n")
	for i := 0; i < numDeps; i++ {
		sb.WriteString(fmt.Sprintf("            nativeImplementation 'lib%d:lib%d:1.0.%d'\n", i, i, i))
	}
	sb.WriteString("        }\n    }\n}")
	return sb.String()
}

func generateLargeNinjaFile(numRules int) string {
	var sb strings.Builder
	sb.WriteString("cxx = g++\ncxxflags = -Wall\n\n")
	for i := 0; i < numRules; i++ {
		sb.WriteString(fmt.Sprintf("build obj%d.o: cxx src%d.cpp | header%d.h\n", i, i, i))
	}
	return sb.String()
}

func generateLargeSConsFile(numTargets int) string {
	var sb strings.Builder
	sb.WriteString("env = Environment()\n\n")
	for i := 0; i < numTargets; i++ {
		sb.WriteString(fmt.Sprintf("lib%d = env.Library('lib%d', ['src%d.cpp'])\n", i, i, i))
	}
	return sb.String()
}

// 注意事项:
// 1. 集成测试验证了多个提取器的协同工作
// 2. 跨项目依赖测试验证了复杂项目结构的处理
// 3. 性能测试验证了大规模项目的处理能力
// 4. 使用临时文件和目录进行测试
// 5. 包含了详细的性能指标和验证

// 运行测试示例:
// 运行所有测试:
// go test -v ./internal/extractor/extractor_integration_test.go
// 
// 仅运行非性能测试:
// go test -v -short ./internal/extractor/extractor_integration_test.go
//
// 运行特定测试:
// go test -v -run TestMultipleExtractors_Integration ./internal/extractor/extractor_integration_test.go 