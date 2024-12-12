package extractor

import (
	"github.com/lkpsg/ccscanner/pkg/models"
)

// Extractor 依赖提取器接口
type Extractor interface {
	// Extract 提取依赖信息
	Extract() ([]models.Dependency, error)
}

// BaseExtractor 基础提取器
type BaseExtractor struct {
	FilePath string // 文件路径
}

// NewBaseExtractor 创建基础提取器
func NewBaseExtractor(path string) BaseExtractor {
	return BaseExtractor{
		FilePath: path,
	}
}

// ExtractorType 提取器类型
type ExtractorType string

const (
	CMakeExtractorType     ExtractorType = "cmake"     // CMake提取器
	MakeExtractorType      ExtractorType = "make"      // Make提取器
	ConanExtractorType     ExtractorType = "conan"     // Conan提取器
	VcpkgExtractorType     ExtractorType = "vcpkg"     // Vcpkg提取器
	SubmoduleExtractorType ExtractorType = "submodule" // Git子模块提取器
	MesonExtractorType     ExtractorType = "meson"     // Meson提取器
	PkgConfigExtractorType ExtractorType = "pkgconfig" // PkgConfig提取器
	AutoconfExtractorType  ExtractorType = "autoconf"  // Autoconf提取器
	ControlExtractorType   ExtractorType = "control"   // Control提取器
)

// ExtractorFactory 提取器工厂
type ExtractorFactory interface {
	// CreateExtractor 创建提取器
	CreateExtractor(path string) Extractor
}

// RegisteredExtractors 已注册的提取器工厂
var RegisteredExtractors = make(map[ExtractorType]ExtractorFactory)

// RegisterExtractor 注册提取器工厂
func RegisterExtractor(typ ExtractorType, factory ExtractorFactory) {
	RegisteredExtractors[typ] = factory
}

// GetExtractor 获取提取器
func GetExtractor(typ ExtractorType, path string) Extractor {
	if factory, ok := RegisteredExtractors[typ]; ok {
		return factory.CreateExtractor(path)
	}
	return nil
}

// ExtractorConfig 提取器配置
type ExtractorConfig struct {
	// 通用配置
	IgnoreComments bool     // 是否忽略注释
	IgnoreTests    bool     // 是否忽略测试依赖
	ExcludeFiles   []string // 排除的文件
	IncludeFiles   []string // 包含的文件
	MaxDepth       int      // 最大递归深度

	// CMake配置
	CMakeFlags []string // CMake标志

	// Make配置
	MakeFlags []string // Make标志

	// Conan配置
	ConanRemotes []string // Conan远程仓库

	// Vcpkg配置
	VcpkgRoot string // Vcpkg根目录

	// Git配置
	GitBranch string // Git分支
}

// DefaultConfig 默认配置
var DefaultConfig = ExtractorConfig{
	IgnoreComments: true,
	IgnoreTests:    false,
	MaxDepth:       10,
}

// ExtractorError 提取器错误
type ExtractorError struct {
	Type    ExtractorType // 提取器类型
	File    string        // 文件路径
	Message string        // 错误信息
}

// Error 实现error接口
func (e ExtractorError) Error() string {
	return fmt.Sprintf("%s extractor error in %s: %s", e.Type, e.File, e.Message)
}

// NewExtractorError 创建提取器错误
func NewExtractorError(typ ExtractorType, file, msg string) ExtractorError {
	return ExtractorError{
		Type:    typ,
		File:    file,
		Message: msg,
	}
}

/*
使用示例:

1. 实现提取器接口:
type CMakeExtractor struct {
    BaseExtractor
}

func (e *CMakeExtractor) Extract() ([]models.Dependency, error) {
    // 实现CMake依赖提取逻辑
    return deps, nil
}

2. 创建提取器工厂:
type CMakeExtractorFactory struct{}

func (f *CMakeExtractorFactory) CreateExtractor(path string) Extractor {
    return &CMakeExtractor{
        BaseExtractor: NewBaseExtractor(path),
    }
}

3. 注册提取器:
RegisterExtractor(CMakeExtractorType, &CMakeExtractorFactory{})

4. 使用提取器:
extractor := GetExtractor(CMakeExtractorType, "CMakeLists.txt")
if extractor != nil {
    deps, err := extractor.Extract()
    if err != nil {
        log.Printf("Failed to extract dependencies: %v\n", err)
    }
    // 处理依赖信息
}
*/ 