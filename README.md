# CCScanner - C/C++项目依赖扫描工具

CCScanner 是一个用于扫描和分析 C/C++ 项目依赖的工具。它支持多种构建系统和包管理器,可以帮助你:

- 扫描项目依赖
- 分析依赖关系
- 检测漏洞风险
- 生成依赖报告

## 功能特点

- 支持多种构建系统:
  - CMake
  - Make
  - Bazel
  - Buck
  - Meson
  - Conan
  - vcpkg
  - pkg-config
  - Autoconf
  - Git Submodules

- 多种输出格式:
  - JSON
  - 文本
  - HTML

- 漏洞检测:
  - 支持多个漏洞数据库
  - 提供漏洞严重程度评估
  - 包含修复建议

- 依赖分析:
  - 构建依赖关系图
  - 检测循环依赖
  - 分析依赖树深度

- 其他特性:
  - 支持缓存加速扫描
  - 提供 Web 界面
  - 支持自定义输出模板
  - 详细的扫描报告

## 安装

### 从源码安装

```bash
git clone https://github.com/yourusername/ccscanner.git
cd ccscanner
go build -o ccscanner cmd/ccscanner/main.go
```

### 使用 Go 工具安装

```bash
go install github.com/yourusername/ccscanner/cmd/ccscanner@latest
```

## 使用方法

### 基本用法

```bash
# 扫描当前目录
ccscanner -path .

# 扫描指定目录
ccscanner -path /path/to/project

# 指定输出格式和文件
ccscanner -path /path/to/project -format json -output result.json

# 详细模式
ccscanner -path /path/to/project -verbose
```

### 高级选项

```bash
# 排除特定目录
ccscanner -path /path/to/project -exclude "vendor/*,third_party/*"

# 限制扫描深度
ccscanner -path /path/to/project -depth 5

# 禁用漏洞扫描
ccscanner -path /path/to/project -vulns=false

# 禁用依赖扫描
ccscanner -path /path/to/project -deps=false
```

### 输出格式

#### JSON 格式

```bash
ccscanner -path /path/to/project -format json -output result.json
```

生成的 JSON 文件包含完整的扫描结果:
```json
{
  "projectPath": "/path/to/project",
  "startTime": "2023-12-11T10:00:00Z",
  "duration": "5s",
  "dependencies": [
    {
      "name": "boost",
      "type": "system",
      "filePath": "CMakeLists.txt",
      "line": 10
    }
  ],
  "vulnerabilities": [
    {
      "id": "CVE-2023-1234",
      "severity": "high",
      "description": "严重的安全漏洞",
      "affectedComponent": "openssl",
      "fixedVersion": "1.1.1t"
    }
  ]
}
```

#### 文本格式

```bash
ccscanner -path /path/to/project -format text -output report.txt
```

生成的文本报告易于阅读:
```
CCScanner 扫描报告
================

项目路径: /path/to/project
扫描时间: 2023-12-11 10:00:00
耗时: 5s

找到 2 个依赖:
- boost (system)
  文件: CMakeLists.txt:10
- openssl (system)
  文件: CMakeLists.txt:15

发现 1 个漏洞:
- CVE-2023-1234 (严重程度: high)
  描述: 严重的安全漏洞
  影响: openssl
  修复版本: 1.1.1t
```

#### HTML 格式

```bash
ccscanner -path /path/to/project -format html -output report.html
```

生成美观的 HTML 报告,包含:
- 项目信息
- 依赖列表
- 漏洞信息
- 依赖关系图
- 统计数据

### 自定义模板

你可以使用自定义模板来生成 HTML 报告:

1. 创建模板文件 `template.html`:
```html
<!DOCTYPE html>
<html>
<head><title>自定义报告</title></head>
<body>
<h1>项目: {{.Result.ProjectPath}}</h1>
<p>依赖数量: {{len .Result.Dependencies}}</p>
</body>
</html>
```

2. 使用自定义模板:
```bash
ccscanner -path /path/to/project -format html -template template.html -output report.html
```

## 配置文件

CCScanner 支持通过配置文件设置默认选项。创建 `.ccscanner.yaml` 文件:

```yaml
exclude:
  - vendor/*
  - third_party/*
depth: 5
format: json
verbose: true
```

## 开发

### 目录结构

```
.
├── cmd/
│   └── ccscanner/          # 命令行工具
├── internal/
│   ├── analyzer/           # 依赖分析器
│   ├── cache/             # 缓存实现
│   ├── extractor/         # 依赖提取器
│   ├── scanner/           # 核心扫描器
│   ├── vulnerability/     # 漏洞检测
│   └── web/              # Web 界面
├── pkg/
│   ├── formatter/         # 输出格式化
│   ├── models/           # 数据模型
│   └── utils/            # 工具函数
└── test/                 # 测试文件
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./pkg/formatter

# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 添加新的提取器

1. 在 `internal/extractor` 目录下创建新文件
2. 实现 `Extractor` 接口
3. 在 `cmd/ccscanner/main.go` 中注册新的提取器

示例:
```go
type NewExtractor struct {
    BaseExtractor
}

func (e *NewExtractor) Extract(projectPath string, filePath string) ([]models.Dependency, error) {
    // 实现提取逻辑
}
```

## 贡献

欢迎贡献代码!请遵循以下步骤:

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 作者

- 作者名字 - [@yourusername](https://github.com/yourusername)

## 致谢

- [Boost](https://www.boost.org/)
- [OpenSSL](https://www.openssl.org/)
- [其他项目...]

