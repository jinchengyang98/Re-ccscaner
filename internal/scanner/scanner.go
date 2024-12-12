package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lkpsg/ccscanner/internal/cache"
	"github.com/lkpsg/ccscanner/internal/extractor"
	"github.com/lkpsg/ccscanner/pkg/models"
	"github.com/lkpsg/ccscanner/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Config 扫描器配置
type Config struct {
	TargetDir    string     // 目标目录
	OutputFile   string     // 输出文件
	EnableCache  bool       // 是否启用缓存
	MaxWorkers   int        // 最大工作协程数
	Logger       *zap.Logger // 日志记录器
}

// Scanner 依赖扫描器
type Scanner struct {
	config Config
	cache  *cache.Cache
	result *models.DependencyResult
	mu     sync.Mutex // 保护result
}

// NewScanner 创建新的扫描器实例
func NewScanner(config Config) *Scanner {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 10 // 默认10个工作协程
	}
	if config.Logger == nil {
		config.Logger, _ = zap.NewProduction()
	}

	scanner := &Scanner{
		config: config,
		result: models.NewDependencyResult(
			filepath.Base(config.TargetDir),
			config.TargetDir,
		),
	}

	if config.EnableCache {
		scanner.cache = cache.NewCache()
	}

	return scanner
}

// Scan 执行依赖扫描
func (s *Scanner) Scan() error {
	startTime := time.Now()
	s.config.Logger.Info("开始扫描",
		zap.String("target", s.config.TargetDir),
		zap.Bool("cache_enabled", s.config.EnableCache),
	)

	// 创建错误组和信号量
	eg := errgroup.Group{}
	sem := make(chan struct{}, s.config.MaxWorkers)

	// 遍历目录
	err := filepath.Walk(s.config.TargetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏文件和目录
		if utils.IsHidden(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 检查是否是配置文件
		ext := s.detectFileType(path, info)
		if ext == nil {
			return nil
		}

		// 如果启用了缓存,检查缓存
		if s.config.EnableCache {
			if deps, ok := s.cache.Get(path); ok {
				s.addDependencies(deps)
				return nil
			}
		}

		// 添加扫描任务
		sem <- struct{}{} // 获取信号量
		eg.Go(func() error {
			defer func() { <-sem }() // 释放信号量

			// 提取依赖
			deps, err := ext.Extract()
			if err != nil {
				s.config.Logger.Error("依赖提取失败",
					zap.String("file", path),
					zap.Error(err),
				)
				s.result.AddError(fmt.Sprintf("Failed to extract dependencies from %s: %v", path, err))
				return nil
			}

			// 更新缓存
			if s.config.EnableCache {
				s.cache.Set(path, deps)
			}

			// 添加依赖
			s.addDependencies(deps)
			return nil
		})

		return nil
	})

	if err != nil {
		return fmt.Errorf("扫描目录失败: %v", err)
	}

	// 等待所有任务完成
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("扫描任务失败: %v", err)
	}

	// 更新扫描时间
	s.result.ScanDuration = time.Since(startTime).Seconds()

	s.config.Logger.Info("扫描完成",
		zap.Int("total_deps", s.result.TotalDeps),
		zap.Int("direct_deps", s.result.DirectDeps),
		zap.Int("indirect_deps", s.result.IndirectDeps),
		zap.Int("vulnerable_deps", s.result.VulnerableDeps),
		zap.Float64("duration", s.result.ScanDuration),
	)

	return nil
}

// SaveResults 保存扫描结果
func (s *Scanner) SaveResults() error {
	// 创建输出目录
	if err := os.MkdirAll(filepath.Dir(s.config.OutputFile), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 序列化结果
	data, err := json.MarshalIndent(s.result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化结果失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(s.config.OutputFile, data, 0644); err != nil {
		return fmt.Errorf("写入结果文件失败: %v", err)
	}

	return nil
}

// GetResults 获取扫描结果
func (s *Scanner) GetResults() *models.DependencyResult {
	return s.result
}

// addDependencies 添加依赖项(线程安全)
func (s *Scanner) addDependencies(deps []models.Dependency) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, dep := range deps {
		s.result.AddDependency(dep)
	}
}

// detectFileType 检测文件类型并返回相应的提取器
func (s *Scanner) detectFileType(path string, info os.FileInfo) extractor.Extractor {
	filename := info.Name()
	ext := filepath.Ext(filename)

	// 根据文件名和扩展名判断文件类型
	switch {
	case filename == "CMakeLists.txt" || ext == ".cmake":
		return extractor.NewCMakeExtractor(path)
	case filename == "Makefile" || filename == "makefile":
		return extractor.NewMakeExtractor(path)
	case filename == "conanfile.txt" || filename == "conanfile.py":
		return extractor.NewConanExtractor(path)
	case filename == "vcpkg.json":
		return extractor.NewVcpkgExtractor(path)
	case filename == ".gitmodules":
		return extractor.NewSubmoduleExtractor(path)
	case filename == "meson.build":
		return extractor.NewMesonExtractor(path)
	case ext == ".pc":
		return extractor.NewPkgConfigExtractor(path)
	case filename == "configure" || filename == "configure.ac":
		return extractor.NewAutoconfExtractor(path)
	case ext == ".dsc":
		return extractor.NewControlExtractor(path)
	}

	return nil
}

/*
使用示例:

1. 创建扫描器:
scanner := NewScanner(Config{
	TargetDir: "/path/to/project",
	OutputFile: "results.json",
	EnableCache: true,
	MaxWorkers: 10,
	Logger: logger,
})

2. 执行扫描:
if err := scanner.Scan(); err != nil {
	log.Fatal(err)
}

3. 保存结果:
if err := scanner.SaveResults(); err != nil {
	log.Fatal(err)
}

4. 获取结果:
results := scanner.GetResults()
*/ 