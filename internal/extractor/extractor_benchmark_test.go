package extractor

import (
	"os"
	"path/filepath"
	"testing"
)

// setupBenchmarkFiles 创建基准测试文件
func setupBenchmarkFiles(b *testing.B, numDeps int) (string, func()) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "ccscanner-benchmark-*")
	if err != nil {
		b.Fatal(err)
	}

	// 创建 Gradle 文件
	gradleContent := generateLargeGradleFile(numDeps)
	gradleFile := filepath.Join(tempDir, "build.gradle")
	if err := os.WriteFile(gradleFile, []byte(gradleContent), 0644); err != nil {
		b.Fatal(err)
	}

	// 创建 Ninja 文件
	ninjaContent := generateLargeNinjaFile(numDeps)
	ninjaFile := filepath.Join(tempDir, "build.ninja")
	if err := os.WriteFile(ninjaFile, []byte(ninjaContent), 0644); err != nil {
		b.Fatal(err)
	}

	// 创建 SCons 文件
	sconsContent := generateLargeSConsFile(numDeps)
	sconsFile := filepath.Join(tempDir, "SConstruct")
	if err := os.WriteFile(sconsFile, []byte(sconsContent), 0644); err != nil {
		b.Fatal(err)
	}

	// 返回清理函数
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// BenchmarkGradleExtractor 测试 Gradle 提取器的性能
func BenchmarkGradleExtractor(b *testing.B) {
	benchmarkSizes := []int{10, 100, 1000}
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("deps=%d", size), func(b *testing.B) {
			tempDir, cleanup := setupBenchmarkFiles(b, size)
			defer cleanup()

			extractor := NewGradleExtractor()
			filePath := filepath.Join(tempDir, "build.gradle")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				deps, err := extractor.Extract(tempDir, filePath)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(deps)), "deps")
			}
		})
	}
}

// BenchmarkNinjaExtractor 测试 Ninja 提取器的性能
func BenchmarkNinjaExtractor(b *testing.B) {
	benchmarkSizes := []int{10, 100, 1000}
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("rules=%d", size), func(b *testing.B) {
			tempDir, cleanup := setupBenchmarkFiles(b, size)
			defer cleanup()

			extractor := NewNinjaExtractor()
			filePath := filepath.Join(tempDir, "build.ninja")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				deps, err := extractor.Extract(tempDir, filePath)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(deps)), "deps")
			}
		})
	}
}

// BenchmarkSconsExtractor 测试 SCons 提取器的性能
func BenchmarkSconsExtractor(b *testing.B) {
	benchmarkSizes := []int{10, 100, 1000}
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("targets=%d", size), func(b *testing.B) {
			tempDir, cleanup := setupBenchmarkFiles(b, size)
			defer cleanup()

			extractor := NewSconsExtractor()
			filePath := filepath.Join(tempDir, "SConstruct")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				deps, err := extractor.Extract(tempDir, filePath)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(deps)), "deps")
			}
		})
	}
}

// BenchmarkMultipleExtractors 测试多个提取器同时工作的性能
func BenchmarkMultipleExtractors(b *testing.B) {
	benchmarkSizes := []int{10, 100, 1000}
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			tempDir, cleanup := setupBenchmarkFiles(b, size)
			defer cleanup()

			extractors := []Extractor{
				NewGradleExtractor(),
				NewNinjaExtractor(),
				NewSconsExtractor(),
			}

			files := []string{
				filepath.Join(tempDir, "build.gradle"),
				filepath.Join(tempDir, "build.ninja"),
				filepath.Join(tempDir, "SConstruct"),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var totalDeps int
				for j, extractor := range extractors {
					deps, err := extractor.Extract(tempDir, files[j])
					if err != nil {
						b.Fatal(err)
					}
					totalDeps += len(deps)
				}
				b.ReportMetric(float64(totalDeps), "total_deps")
			}
		})
	}
}

// BenchmarkExtractorMemory 测试提取器的内存使用
func BenchmarkExtractorMemory(b *testing.B) {
	size := 1000 // 使用较大的文件来测试内存使用
	tempDir, cleanup := setupBenchmarkFiles(b, size)
	defer cleanup()

	extractors := []struct {
		name      string
		extractor Extractor
		file      string
	}{
		{
			name:      "Gradle",
			extractor: NewGradleExtractor(),
			file:      filepath.Join(tempDir, "build.gradle"),
		},
		{
			name:      "Ninja",
			extractor: NewNinjaExtractor(),
			file:      filepath.Join(tempDir, "build.ninja"),
		},
		{
			name:      "SCons",
			extractor: NewSconsExtractor(),
			file:      filepath.Join(tempDir, "SConstruct"),
		},
	}

	for _, e := range extractors {
		b.Run(e.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				deps, err := e.extractor.Extract(tempDir, e.file)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(deps)), "deps")
			}
		})
	}
}

// BenchmarkParallelExtraction 测试并行提取的性能
func BenchmarkParallelExtraction(b *testing.B) {
	size := 100
	tempDir, cleanup := setupBenchmarkFiles(b, size)
	defer cleanup()

	extractors := []struct {
		name      string
		extractor Extractor
		file      string
	}{
		{
			name:      "Gradle",
			extractor: NewGradleExtractor(),
			file:      filepath.Join(tempDir, "build.gradle"),
		},
		{
			name:      "Ninja",
			extractor: NewNinjaExtractor(),
			file:      filepath.Join(tempDir, "build.ninja"),
		},
		{
			name:      "SCons",
			extractor: NewSconsExtractor(),
			file:      filepath.Join(tempDir, "SConstruct"),
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for _, e := range extractors {
				deps, err := e.extractor.Extract(tempDir, e.file)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(deps)), e.name+"_deps")
			}
		}
	})
}

// 注意事项:
// 1. 基准测试包含了不同规模的输入
// 2. 测试了单个提取器和多个提取器的性能
// 3. 包含了内存使用和分配的测试
// 4. 测试了并行提取的性能
// 5. 使用临时文件和目录进行测试

// 运行基准测试示例:
// 运行所有基准测试:
// go test -bench=. ./internal/extractor/extractor_benchmark_test.go
//
// 运行特定基准测试:
// go test -bench=BenchmarkGradleExtractor ./internal/extractor/extractor_benchmark_test.go
//
// 运行基准测试并生成内存分析:
// go test -bench=. -benchmem ./internal/extractor/extractor_benchmark_test.go
//
// 运行基准测试并生成 CPU 分析:
// go test -bench=. -cpuprofile=cpu.prof ./internal/extractor/extractor_benchmark_test.go 