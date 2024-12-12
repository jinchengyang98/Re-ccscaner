package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lkpsg/ccscanner/pkg/models"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Hash       string             `json:"hash"`       // 文件内容哈希
	UpdateTime time.Time          `json:"updateTime"` // 更新时间
	Deps       []models.Dependency `json:"deps"`      // 依赖信息
}

// Cache 缓存管理器
type Cache struct {
	entries map[string]CacheEntry // 缓存条目映射
	mu      sync.RWMutex         // 读写锁
	dir     string               // 缓存目录
}

// NewCache 创建新的缓存管理器
func NewCache() *Cache {
	// 获取缓存目录
	cacheDir := getCacheDir()

	cache := &Cache{
		entries: make(map[string]CacheEntry),
		dir:     cacheDir,
	}

	// 加载持久化的缓存
	cache.load()

	return cache
}

// Get 获取缓存
func (c *Cache) Get(path string) ([]models.Dependency, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 计算文件哈希
	hash, err := hashFile(path)
	if err != nil {
		return nil, false
	}

	// 查找缓存
	entry, ok := c.entries[path]
	if !ok {
		return nil, false
	}

	// 验证哈希是否匹配
	if entry.Hash != hash {
		return nil, false
	}

	// 检查缓存是否过期(7天)
	if time.Since(entry.UpdateTime) > 7*24*time.Hour {
		return nil, false
	}

	return entry.Deps, true
}

// Set 设置缓存
func (c *Cache) Set(path string, deps []models.Dependency) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 计算文件哈希
	hash, err := hashFile(path)
	if err != nil {
		return err
	}

	// 更新缓存
	c.entries[path] = CacheEntry{
		Hash:       hash,
		UpdateTime: time.Now(),
		Deps:       deps,
	}

	// 持久化缓存
	return c.save()
}

// Clear 清除缓存
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]CacheEntry)
	return os.RemoveAll(c.dir)
}

// load 从磁盘加载缓存
func (c *Cache) load() error {
	// 确保缓存目录存在
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}

	// 读取缓存文件
	cacheFile := filepath.Join(c.dir, "cache.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// 解析缓存数据
	return json.Unmarshal(data, &c.entries)
}

// save 将缓存保存到磁盘
func (c *Cache) save() error {
	// 确保缓存目录存在
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}

	// 序列化缓存数据
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return err
	}

	// 写入缓存文件
	cacheFile := filepath.Join(c.dir, "cache.json")
	return os.WriteFile(cacheFile, data, 0644)
}

// hashFile 计算文件内容的SHA256哈希
func hashFile(path string) (string, error) {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 创建哈希对象
	hash := sha256.New()

	// 计算文件内容哈希
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// 返回十六进制哈希字符串
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// getCacheDir 获取缓存目录
func getCacheDir() string {
	// 获取用户缓存目录
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// 如果获取失败,使用临时目录
		cacheDir = os.TempDir()
	}

	// 返回ccscanner的缓存目录
	return filepath.Join(cacheDir, "ccscanner")
}

// GetCacheStats 获取缓存统计信息
func (c *Cache) GetCacheStats() (int, int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := len(c.entries)
	size := int64(0)

	// 计算缓存大小
	cacheFile := filepath.Join(c.dir, "cache.json")
	if info, err := os.Stat(cacheFile); err == nil {
		size = info.Size()
	}

	return count, size
}

// RemoveExpired 移除过期的缓存条目
func (c *Cache) RemoveExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	now := time.Now()

	// 遍历所有缓存条目
	for path, entry := range c.entries {
		// 如果缓存超过7天,删除它
		if now.Sub(entry.UpdateTime) > 7*24*time.Hour {
			delete(c.entries, path)
			count++
		}
	}

	// 如果有条目被删除,保存更新后的缓存
	if count > 0 {
		c.save()
	}

	return count
}

// Validate 验证缓存的完整性
func (c *Cache) Validate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path, entry := range c.entries {
		// 检查文件是否存在
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delete(c.entries, path)
			continue
		}

		// 验证文件哈希
		hash, err := hashFile(path)
		if err != nil {
			return fmt.Errorf("failed to hash file %s: %v", path, err)
		}

		if hash != entry.Hash {
			delete(c.entries, path)
		}
	}

	return c.save()
}

/*
使用示例:

1. 创建缓存:
cache := NewCache()

2. 获取缓存:
if deps, ok := cache.Get(filePath); ok {
    // 使用缓存的依赖信息
    useCachedDeps(deps)
} else {
    // 缓存未命中,需要重新扫描
    deps := scanDependencies(filePath)
    cache.Set(filePath, deps)
}

3. 获取缓存统计:
count, size := cache.GetCacheStats()
fmt.Printf("Cache entries: %d, size: %d bytes\n", count, size)

4. 清理过期缓存:
removed := cache.RemoveExpired()
fmt.Printf("Removed %d expired cache entries\n", removed)

5. 验证缓存:
if err := cache.Validate(); err != nil {
    log.Printf("Cache validation failed: %v\n", err)
}

6. 清除所有缓存:
if err := cache.Clear(); err != nil {
    log.Printf("Failed to clear cache: %v\n", err)
}
*/ 