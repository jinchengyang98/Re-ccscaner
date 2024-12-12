package models

import "time"

// Dependency 表示一个依赖项
type Dependency struct {
	// 基本信息
	Name        string   `json:"name"`        // 依赖名称
	Version     string   `json:"version"`     // 版本号
	Type        string   `json:"type"`        // 依赖类型(如: library, framework等)
	Description string   `json:"description"` // 描述信息
	Homepage    string   `json:"homepage"`    // 主页
	License     string   `json:"license"`     // 许可证
	Languages   []string `json:"languages"`   // 编程语言

	// 来源信息
	Source     string `json:"source"`      // 来源(如: github, gitlab等)
	Repository string `json:"repository"`   // 仓库地址
	Branch     string `json:"branch"`      // 分支
	Commit     string `json:"commit"`      // 提交hash

	// 依赖关系
	Dependencies   []string          `json:"dependencies"`    // 直接依赖
	DevDependencies []string         `json:"devDependencies"` // 开发依赖
	Conflicts      []string          `json:"conflicts"`       // 冲突项
	Optional       bool              `json:"optional"`        // 是否可选
	Required       bool              `json:"required"`        // 是否必需
	Constraints    []VersionConstrain `json:"constraints"`    // 版本约束

	// 构建信息
	BuildSystem    string   `json:"buildSystem"`    // 构建系统(如: cmake, make等)
	BuildFlags     []string `json:"buildFlags"`     // 构建标志
	BuildCommands  []string `json:"buildCommands"`  // 构建命令
	InstallCommands []string `json:"installCommands"` // 安装命令

	// 安全信息
	Vulnerabilities []Vulnerability `json:"vulnerabilities"` // 漏洞信息
	SecurityScore   float64        `json:"securityScore"`   // 安全评分
	LastAudit       time.Time      `json:"lastAudit"`      // 最后审计时间

	// 元数据
	DetectedBy     string    `json:"detectedBy"`     // 检测工具
	DetectedAt     time.Time `json:"detectedAt"`     // 检测时间
	LastUpdated    time.Time `json:"lastUpdated"`    // 最后更新时间
	ConfigFile     string    `json:"configFile"`     // 配置文件路径
	ConfigFileType string    `json:"configFileType"` // 配置文件类型
}

// VersionConstrain 表示版本约束
type VersionConstrain struct {
	Operator string `json:"operator"` // 操作符(如: >=, <=, =等)
	Version  string `json:"version"`  // 版本号
}

// Vulnerability 表示漏洞信息
type Vulnerability struct {
	ID          string    `json:"id"`          // 漏洞ID
	Title       string    `json:"title"`       // 标题
	Description string    `json:"description"` // 描述
	Severity    string    `json:"severity"`    // 严重程度
	CVSS        float64   `json:"cvss"`       // CVSS评分
	Published   time.Time `json:"published"`   // 发布时间
	Fixed       bool      `json:"fixed"`       // 是否已修复
	FixedIn     string    `json:"fixedIn"`     // 修复版本
	References  []string  `json:"references"`  // 参考链接
}

// DependencyResult 表示依赖扫描结果
type DependencyResult struct {
	// 项目信息
	ProjectName    string    `json:"projectName"`    // 项目名称
	ProjectPath    string    `json:"projectPath"`    // 项目路径
	ScanTime      time.Time `json:"scanTime"`       // 扫描时间
	ScanDuration  float64   `json:"scanDuration"`   // 扫描耗时(秒)
	
	// 统计信息
	TotalDeps     int `json:"totalDeps"`     // 总依赖数
	DirectDeps    int `json:"directDeps"`    // 直接依赖数
	IndirectDeps  int `json:"indirectDeps"`  // 间接依赖数
	VulnerableDeps int `json:"vulnerableDeps"` // 存在漏洞的依赖数
	
	// 依赖列表
	Dependencies []Dependency `json:"dependencies"` // 依赖列表
	
	// 构建系统信息
	BuildSystems []string `json:"buildSystems"` // 使用的构建系统
	
	// 错误信息
	Errors []string `json:"errors"` // 扫描过程中的错误
}

// NewDependency 创建新的依赖项
func NewDependency(name string) *Dependency {
	return &Dependency{
		Name:       name,
		DetectedAt: time.Now(),
		Required:   true,
		Languages:  []string{"C", "C++"},
	}
}

// NewDependencyResult 创建新的依赖扫描结果
func NewDependencyResult(projectName, projectPath string) *DependencyResult {
	return &DependencyResult{
		ProjectName:   projectName,
		ProjectPath:   projectPath,
		ScanTime:     time.Now(),
		Dependencies: make([]Dependency, 0),
		BuildSystems: make([]string, 0),
		Errors:      make([]string, 0),
	}
}

// AddDependency 添加依赖项
func (r *DependencyResult) AddDependency(dep Dependency) {
	r.Dependencies = append(r.Dependencies, dep)
	r.TotalDeps++
	if len(dep.Dependencies) == 0 {
		r.DirectDeps++
	} else {
		r.IndirectDeps++
	}
	if len(dep.Vulnerabilities) > 0 {
		r.VulnerableDeps++
	}
}

// AddError 添加错误信息
func (r *DependencyResult) AddError(err string) {
	r.Errors = append(r.Errors, err)
}

// AddBuildSystem 添加构建系统
func (r *DependencyResult) AddBuildSystem(system string) {
	for _, s := range r.BuildSystems {
		if s == system {
			return
		}
	}
	r.BuildSystems = append(r.BuildSystems, system)
}

/*
使用示例:

1. 创建新的依赖项:
dep := NewDependency("boost")
dep.Version = "1.76.0"
dep.Type = "library"
dep.Description = "Boost C++ Libraries"
dep.License = "BSL-1.0"

2. 创建扫描结果:
result := NewDependencyResult("myproject", "/path/to/myproject")

3. 添加依赖:
result.AddDependency(dep)

4. 添加构建系统:
result.AddBuildSystem("cmake")

5. 添加错误:
result.AddError("Failed to parse CMakeLists.txt")
*/ 