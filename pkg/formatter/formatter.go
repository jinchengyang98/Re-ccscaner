package formatter

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/yourusername/ccscanner/pkg/models"
)

// Formatter 定义了格式化器的接口
type Formatter interface {
	Format(result *models.ScanResult) ([]byte, error)
}

// JSONFormatter JSON格式化器
type JSONFormatter struct {
	PrettyPrint bool
}

// TextFormatter 文本格式化器
type TextFormatter struct {
	Verbose bool
}

// HTMLFormatter HTML格式化器
type HTMLFormatter struct {
	Template string
}

// NewFormatter 创建一个新的格式化器
func NewFormatter(format string, options map[string]interface{}) (Formatter, error) {
	switch strings.ToLower(format) {
	case "json":
		prettyPrint := false
		if v, ok := options["pretty"]; ok {
			prettyPrint = v.(bool)
		}
		return &JSONFormatter{PrettyPrint: prettyPrint}, nil
	case "text":
		verbose := false
		if v, ok := options["verbose"]; ok {
			verbose = v.(bool)
		}
		return &TextFormatter{Verbose: verbose}, nil
	case "html":
		template := ""
		if v, ok := options["template"]; ok {
			template = v.(string)
		}
		return &HTMLFormatter{Template: template}, nil
	default:
		return nil, fmt.Errorf("不支持的格式: %s", format)
	}
}

// Format 实现 JSON 格式化
func (f *JSONFormatter) Format(result *models.ScanResult) ([]byte, error) {
	if f.PrettyPrint {
		return json.MarshalIndent(result, "", "  ")
	}
	return json.Marshal(result)
}

// Format 实现文本格式化
func (f *TextFormatter) Format(result *models.ScanResult) ([]byte, error) {
	var b strings.Builder

	// 写入标题
	b.WriteString("CCScanner 扫描报告\n")
	b.WriteString(strings.Repeat("=", 80) + "\n\n")

	// 基本信息
	b.WriteString(fmt.Sprintf("项目路径: %s\n", result.ProjectPath))
	b.WriteString(fmt.Sprintf("扫描时间: %s\n", result.StartTime.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("耗时: %s\n\n", result.Duration))

	// 依赖信息
	b.WriteString(fmt.Sprintf("找到 %d 个依赖:\n", len(result.Dependencies)))
	if f.Verbose {
		for _, dep := range result.Dependencies {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", dep.Name, dep.Type))
			b.WriteString(fmt.Sprintf("  文件: %s:%d\n", dep.FilePath, dep.Line))
			if dep.Parent != "" {
				b.WriteString(fmt.Sprintf("  父节点: %s\n", dep.Parent))
			}
		}
	} else {
		for _, dep := range result.Dependencies {
			b.WriteString(fmt.Sprintf("- %s\n", dep.Name))
		}
	}
	b.WriteString("\n")

	// 漏洞信息
	if len(result.Vulnerabilities) > 0 {
		b.WriteString(fmt.Sprintf("发现 %d 个漏洞:\n", len(result.Vulnerabilities)))
		for _, vuln := range result.Vulnerabilities {
			b.WriteString(fmt.Sprintf("- %s (严重程度: %s)\n", vuln.ID, vuln.Severity))
			if f.Verbose {
				b.WriteString(fmt.Sprintf("  描述: %s\n", vuln.Description))
				b.WriteString(fmt.Sprintf("  影响: %s\n", vuln.AffectedComponent))
				if vuln.FixedVersion != "" {
					b.WriteString(fmt.Sprintf("  修复版本: %s\n", vuln.FixedVersion))
				}
			}
		}
		b.WriteString("\n")
	}

	// 依赖图信息
	if result.DependencyGraph != nil && f.Verbose {
		b.WriteString("依赖关系图:\n")
		// TODO: 实现依赖图的文本表示
		b.WriteString("(依赖图的详细信息请使用 HTML 格式查看)\n\n")
	}

	return []byte(b.String()), nil
}

// Format 实现 HTML 格式化
func (f *HTMLFormatter) Format(result *models.ScanResult) ([]byte, error) {
	// 使用默认模板或加载自定义模板
	tmpl := defaultHTMLTemplate
	if f.Template != "" {
		var err error
		tmpl, err = template.ParseFiles(f.Template)
		if err != nil {
			return nil, fmt.Errorf("解析模板失败: %v", err)
		}
	}

	// 准备模板数据
	data := struct {
		Result    *models.ScanResult
		Timestamp string
	}{
		Result:    result,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 渲染模板
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return nil, fmt.Errorf("渲染模板失败: %v", err)
	}

	return []byte(b.String()), nil
}

// SaveToFile 将格式化后的结果保存到文件
func SaveToFile(data []byte, filePath string) error {
	return os.WriteFile(filePath, data, 0644)
}

// 默认的 HTML 模板
var defaultHTMLTemplate = template.Must(template.New("report").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>CCScanner 扫描报告</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        h1, h2 {
            color: #333;
        }
        .section {
            margin-bottom: 30px;
            padding: 20px;
            background: #f5f5f5;
            border-radius: 5px;
        }
        .dependency {
            margin: 10px 0;
            padding: 10px;
            background: #fff;
            border: 1px solid #ddd;
            border-radius: 3px;
        }
        .vulnerability {
            margin: 10px 0;
            padding: 10px;
            background: #fff;
            border: 1px solid #ddd;
            border-radius: 3px;
        }
        .vulnerability.high {
            border-left: 5px solid #dc3545;
        }
        .vulnerability.medium {
            border-left: 5px solid #ffc107;
        }
        .vulnerability.low {
            border-left: 5px solid #28a745;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            color: #666;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>CCScanner 扫描报告</h1>
        
        <div class="section">
            <h2>基本信息</h2>
            <p>项目路径: {{.Result.ProjectPath}}</p>
            <p>扫描时间: {{.Result.StartTime.Format "2006-01-02 15:04:05"}}</p>
            <p>耗时: {{.Result.Duration}}</p>
        </div>

        <div class="section">
            <h2>依赖信息</h2>
            <p>共找到 {{len .Result.Dependencies}} 个依赖</p>
            {{range .Result.Dependencies}}
            <div class="dependency">
                <h3>{{.Name}}</h3>
                <p>类型: {{.Type}}</p>
                <p>文件: {{.FilePath}}:{{.Line}}</p>
                {{if .Parent}}
                <p>父节点: {{.Parent}}</p>
                {{end}}
            </div>
            {{end}}
        </div>

        {{if .Result.Vulnerabilities}}
        <div class="section">
            <h2>漏洞信息</h2>
            <p>共发现 {{len .Result.Vulnerabilities}} 个漏洞</p>
            {{range .Result.Vulnerabilities}}
            <div class="vulnerability {{.Severity}}">
                <h3>{{.ID}}</h3>
                <p>严重程度: {{.Severity}}</p>
                <p>描述: {{.Description}}</p>
                <p>影响组件: {{.AffectedComponent}}</p>
                {{if .FixedVersion}}
                <p>修复版本: {{.FixedVersion}}</p>
                {{end}}
            </div>
            {{end}}
        </div>
        {{end}}

        <div class="footer">
            <p>报告生成时间: {{.Timestamp}}</p>
            <p>由 CCScanner 生成</p>
        </div>
    </div>
</body>
</html>
`))

// 注意事项:
// 1. 支持多种输出格式(JSON、文本、HTML)
// 2. 提供了格式化选项(美化、详细程度等)
// 3. HTML输出支持自定义模板
// 4. 文本输出支持详细和简略两种模式
// 5. 提供了保存到文件的功能

// 使用示例:
// formatter, err := NewFormatter("json", map[string]interface{}{"pretty": true})
// if err != nil {
//     log.Fatal(err)
// }
// data, err := formatter.Format(result)
// if err != nil {
//     log.Fatal(err)
// }
// if err := SaveToFile(data, "report.json"); err != nil {
//     log.Fatal(err)
// }
</rewritten_file> 