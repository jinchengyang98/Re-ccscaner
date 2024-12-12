package formatter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/ccscanner/pkg/models"
)

// 创建测试用的扫描结果
func createTestResult() *models.ScanResult {
	return &models.ScanResult{
		ProjectPath: "/path/to/project",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Second * 10),
		Duration:   time.Second * 10,
		Dependencies: []models.Dependency{
			{
				Name:     "boost",
				Type:     "system",
				FilePath: "CMakeLists.txt",
				Line:     10,
				Parent:   "main",
			},
			{
				Name:     "openssl",
				Type:     "system",
				FilePath: "CMakeLists.txt",
				Line:     15,
			},
		},
		Vulnerabilities: []models.Vulnerability{
			{
				ID:               "CVE-2023-1234",
				Severity:         "high",
				Description:      "严重的安全漏洞",
				AffectedComponent: "openssl",
				FixedVersion:     "1.1.1t",
			},
			{
				ID:               "CVE-2023-5678",
				Severity:         "medium",
				Description:      "中等严重性漏洞",
				AffectedComponent: "boost",
				FixedVersion:     "1.81.0",
			},
		},
	}
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		options map[string]interface{}
		wantErr bool
	}{
		{
			name:    "JSON formatter",
			format:  "json",
			options: map[string]interface{}{"pretty": true},
			wantErr: false,
		},
		{
			name:    "Text formatter",
			format:  "text",
			options: map[string]interface{}{"verbose": true},
			wantErr: false,
		},
		{
			name:    "HTML formatter",
			format:  "html",
			options: nil,
			wantErr: false,
		},
		{
			name:    "Invalid format",
			format:  "invalid",
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFormatter(tt.format, tt.options)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, f)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, f)
			}
		})
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	result := createTestResult()

	tests := []struct {
		name        string
		prettyPrint bool
	}{
		{
			name:        "Pretty printed JSON",
			prettyPrint: true,
		},
		{
			name:        "Compact JSON",
			prettyPrint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &JSONFormatter{PrettyPrint: tt.prettyPrint}
			data, err := f.Format(result)
			assert.NoError(t, err)
			assert.NotEmpty(t, data)

			// 验证 JSON 格式是否正确
			var decoded models.ScanResult
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)

			// 验证内容是否正确
			assert.Equal(t, result.ProjectPath, decoded.ProjectPath)
			assert.Equal(t, len(result.Dependencies), len(decoded.Dependencies))
			assert.Equal(t, len(result.Vulnerabilities), len(decoded.Vulnerabilities))
		})
	}
}

func TestTextFormatter_Format(t *testing.T) {
	result := createTestResult()

	tests := []struct {
		name    string
		verbose bool
		checks  []string
	}{
		{
			name:    "Verbose text output",
			verbose: true,
			checks: []string{
				"CCScanner 扫描报告",
				"项目路径: /path/to/project",
				"boost (system)",
				"文件: CMakeLists.txt:10",
				"父节点: main",
				"CVE-2023-1234 (严重程度: high)",
				"描述: 严重的安全漏洞",
			},
		},
		{
			name:    "Simple text output",
			verbose: false,
			checks: []string{
				"CCScanner 扫描报告",
				"项目路径: /path/to/project",
				"- boost",
				"- openssl",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &TextFormatter{Verbose: tt.verbose}
			data, err := f.Format(result)
			assert.NoError(t, err)
			assert.NotEmpty(t, data)

			text := string(data)
			for _, check := range tt.checks {
				assert.Contains(t, text, check)
			}
		})
	}
}

func TestHTMLFormatter_Format(t *testing.T) {
	result := createTestResult()

	tests := []struct {
		name     string
		template string
		checks   []string
	}{
		{
			name:     "Default template",
			template: "",
			checks: []string{
				"<!DOCTYPE html>",
				"CCScanner 扫描报告",
				"项目路径: /path/to/project",
				"boost",
				"openssl",
				"CVE-2023-1234",
				"严重的安全漏洞",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &HTMLFormatter{Template: tt.template}
			data, err := f.Format(result)
			assert.NoError(t, err)
			assert.NotEmpty(t, data)

			html := string(data)
			for _, check := range tt.checks {
				assert.Contains(t, html, check)
			}
		})
	}
}

func TestSaveToFile(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// 测试数据
	testData := []byte("Test content")

	// 保存文件
	err := SaveToFile(testData, testFile)
	assert.NoError(t, err)

	// 验证文件内容
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testData, content)

	// 测试无效路径
	err = SaveToFile(testData, "/invalid/path/test.txt")
	assert.Error(t, err)
}

func TestHTMLFormatter_Format_CustomTemplate(t *testing.T) {
	result := createTestResult()

	// 创建临时目录
	tempDir := t.TempDir()
	templateFile := filepath.Join(tempDir, "custom.html")

	// 创建自定义模板
	customTemplate := `
<!DOCTYPE html>
<html>
<head><title>Custom Report</title></head>
<body>
<h1>Custom Report for {{.Result.ProjectPath}}</h1>
<p>Dependencies: {{len .Result.Dependencies}}</p>
</body>
</html>
`
	err := os.WriteFile(templateFile, []byte(customTemplate), 0644)
	assert.NoError(t, err)

	// 使用自定义模板
	f := &HTMLFormatter{Template: templateFile}
	data, err := f.Format(result)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	html := string(data)
	assert.Contains(t, html, "Custom Report for")
	assert.Contains(t, html, result.ProjectPath)
}

// 注意事项:
// 1. 测试覆盖了所有格式化器类型
// 2. 验证了不同的格式化选项
// 3. 检查了输出内容的正确性
// 4. 包含了错误处理的测试
// 5. 使用临时文件和目录进行测试

// 运行测试示例:
// go test -v ./pkg/formatter 