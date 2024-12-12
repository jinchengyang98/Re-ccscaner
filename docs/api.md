# CCScanner API 文档

本文档详细说明了 CCScanner 的 API 使用方法。

## 目录

- [依赖提取器 API](#依赖提取器-api)
- [依赖分析器 API](#依赖分析器-api)
- [漏洞检测器 API](#漏洞检测器-api)
- [格式化器 API](#格式化器-api)
- [工具函数 API](#工具函数-api)

## 依赖提取器 API

### Extractor 接口

```go
type Extractor interface {
    Extract(projectPath string, filePath string) ([]models.Dependency, error)
    GetName() string
    IsApplicable(filePath string) bool
    GetPriority() int
    GetFilePattern() *regexp.Regexp
}
```

#### 方法说明

- `Extract`: 从指定文件中提取依赖信息
  - 参数:
    - `projectPath`: 项目根目录路径
    - `filePath`: 要分析的文件路径
  - 返回:
    - `[]models.Dependency`: 依赖列表
    - `error`: 错误信息

- `GetName`: 获取提取器名称
  - 返回: 提取器的名称字符串

- `IsApplicable`: 检查提取器是否适用于指定文件
  - 参数:
    - `filePath`: 文件路径
  - 返回: 是否适用的布尔值

- `GetPriority`: 获取提取器优先级
  - 返回: 优先级数值,数值越大优先级越高

- `GetFilePattern`: 获取文件匹配模式
  - 返回: 用于匹配文件名的正则表达式

### 使用示例

```go
// 创建提取器
extractor := NewGradleExtractor()

// 检查文件是否适用
if extractor.IsApplicable("build.gradle") {
    // 提取依赖
    deps, err := extractor.Extract("/path/to/project", "build.gradle")
    if err != nil {
        log.Fatal(err)
    }

    // 处理依赖信息
    for _, dep := range deps {
        fmt.Printf("Found dependency: %s (%s)\n", dep.Name, dep.Type)
    }
}
```

### 支持的提取器

1. Gradle 提取器
   ```go
   extractor := NewGradleExtractor()
   ```
   支持的文件:
   - build.gradle
   - build.gradle.kts
   - settings.gradle
   - settings.gradle.kts

2. Ninja 提取器
   ```go
   extractor := NewNinjaExtractor()
   ```
   支持的文件:
   - build.ninja
   - *.ninja

3. SCons 提取器
   ```go
   extractor := NewSconsExtractor()
   ```
   支持的文件:
   - SConstruct
   - SConscript
   - *.scons

## 依赖分析器 API

### Analyzer 接口

```go
type Analyzer interface {
    Analyze(deps []models.Dependency) (*models.AnalysisResult, error)
    GetName() string
}
```

#### 方法说明

- `Analyze`: 分析依赖关系
  - 参数:
    - `deps`: 依赖列表
  - 返回:
    - `*models.AnalysisResult`: 分析结果
    - `error`: 错误信息

- `GetName`: 获取分析器名称
  - 返回: 分析器的名称字符串

### 使用示例

```go
// 创建分析器
analyzer := NewDependencyAnalyzer()

// 分析依赖
result, err := analyzer.Analyze(deps)
if err != nil {
    log.Fatal(err)
}

// 处理分析结果
fmt.Printf("Found %d circular dependencies\n", len(result.CircularDeps))
fmt.Printf("Maximum dependency depth: %d\n", result.MaxDepth)
```

## 漏洞检测器 API

### Detector 接口

```go
type Detector interface {
    Detect(deps []models.Dependency) ([]models.Vulnerability, error)
    GetName() string
}
```

#### 方法说明

- `Detect`: 检测依赖中的漏洞
  - 参数:
    - `deps`: 依赖列表
  - 返回:
    - `[]models.Vulnerability`: 漏洞列表
    - `error`: 错误信息

- `GetName`: 获取检测器名称
  - 返回: 检测器的名称字符串

### 使用示例

```go
// 创建检测器
detector := NewVulnerabilityDetector()

// 检测漏洞
vulns, err := detector.Detect(deps)
if err != nil {
    log.Fatal(err)
}

// 处理漏洞信息
for _, vuln := range vulns {
    fmt.Printf("Found vulnerability: %s (severity: %s)\n",
        vuln.ID, vuln.Severity)
}
```

## 格式化器 API

### Formatter 接口

```go
type Formatter interface {
    Format(result *models.ScanResult) ([]byte, error)
    GetName() string
}
```

#### 方法说明

- `Format`: 格式化扫描结果
  - 参数:
    - `result`: 扫描结果
  - 返回:
    - `[]byte`: 格式化后的数据
    - `error`: 错误信息

- `GetName`: 获取格式化器名称
  - 返回: 格式化器的名称字符串

### 使用示例

```go
// 创建格式化器
formatter := NewJSONFormatter()

// 格式化结果
data, err := formatter.Format(result)
if err != nil {
    log.Fatal(err)
}

// 保存结果
err = os.WriteFile("result.json", data, 0644)
if err != nil {
    log.Fatal(err)
}
```

## 工具函数 API

### 文件操作

```go
// 检查文件是否存在
func FileExists(path string) bool

// 获取文件扩展名
func GetFileExtension(path string) string

// 获取相对路径
func GetRelativePath(basePath, path string) string
```

### 依赖处理

```go
// 合并依赖列表
func MergeDependencies(deps [][]models.Dependency) []models.Dependency

// 过滤重复依赖
func DeduplicateDependencies(deps []models.Dependency) []models.Dependency

// 排序依赖列表
func SortDependencies(deps []models.Dependency) []models.Dependency
```

### 使用示例

```go
// 检查文件
if utils.FileExists("build.gradle") {
    // 获取扩展名
    ext := utils.GetFileExtension("build.gradle")
    fmt.Printf("File extension: %s\n", ext)

    // 获取相对路径
    relPath := utils.GetRelativePath("/path/to/project", "/path/to/project/src/main.cpp")
    fmt.Printf("Relative path: %s\n", relPath)
}

// 处理依赖
deps = utils.DeduplicateDependencies(deps)
deps = utils.SortDependencies(deps)
```

## 错误处理

所有的 API 都遵循 Go 语言的错误处理惯例,返回错误作为最后一个返回值。建议始终检查错误:

```go
result, err := function()
if err != nil {
    // 处理错误
    log.Printf("Error: %v\n", err)
    return err
}
```

## 配置选项

大多数 API 都支持通过配置选项自定义行为:

```go
// 提取器配置
config := &ExtractorConfig{
    IgnoreComments: true,
    IgnoreTests:    false,
    MaxDepth:       10,
}

// 分析器配置
config := &AnalyzerConfig{
    IgnoreOptional: true,
    MaxDepth:       5,
}

// 检测器配置
config := &DetectorConfig{
    MinSeverity: "high",
    IgnoreDev:   true,
}
```

## 最佳实践

1. 错误处理
   - 始终检查错误返回值
   - 提供有意义的错误信息
   - 使用错误包装添加上下文

2. 资源管理
   - 使用 defer 关闭文件和其他资源
   - 注意内存使用,避免大量分配

3. 并发处理
   - 使用 goroutine 池处理大量文件
   - 注意并发安全性
   - 适当使用同步原语

4. 性能优化
   - 使用缓存减少重复计算
   - 避免不必要的内存分配
   - 使用 strings.Builder 拼接字符串

## 示例项目

完整的示例项目可以在 [examples](../examples) 目录中找到:

- [基本用法](../examples/basic)
- [自定义提取器](../examples/custom-extractor)
- [Web 界面](../examples/web)
- [CI/CD 集成](../examples/ci-cd)

## 常见问题

1. 如何处理大型项目?
   ```go
   // 使用并发处理
   results := make(chan []models.Dependency)
   for _, file := range files {
       go func(f string) {
           deps, _ := extractor.Extract(projectPath, f)
           results <- deps
       }(file)
   }
   ```

2. 如何自定义输出格式?
   ```go
   // 实现 Formatter 接口
   type CustomFormatter struct{}

   func (f *CustomFormatter) Format(result *models.ScanResult) ([]byte, error) {
       // 实现自定义格式化逻辑
   }
   ```

3. 如何扩展支持新的构建系统?
   ```go
   // 实现 Extractor 接口
   type CustomExtractor struct {
       BaseExtractor
   }

   func (e *CustomExtractor) Extract(projectPath, filePath string) ([]models.Dependency, error) {
       // 实现自定义提取逻辑
   }
   ```

## 更多信息

- [项目主页](https://github.com/yourusername/ccscanner)
- [问题追踪](https://github.com/yourusername/ccscanner/issues)
- [贡献指南](CONTRIBUTING.md)
- [更新日志](CHANGELOG.md) 