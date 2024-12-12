package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// IsHidden 检查文件或目录是否是隐藏的
func IsHidden(name string) bool {
	// 忽略以.开头的文件和目录
	if strings.HasPrefix(name, ".") {
		return true
	}

	// Windows系统特殊处理
	if runtime.GOOS == "windows" {
		fileInfo, err := os.Stat(name)
		if err != nil {
			return false
		}
		return fileInfo.Mode()&os.ModeHidden != 0
	}

	return false
}

// DirExists 检查目录是否存在
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetFileSize 获取文件大小
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetFileModTime 获取文件修改时间
func GetFileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// CreateDirIfNotExists 如果目录不存在则创建
func CreateDirIfNotExists(path string) error {
	if !DirExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// RemoveFileIfExists 如果文件存在则删除
func RemoveFileIfExists(path string) error {
	if FileExists(path) {
		return os.Remove(path)
	}
	return nil
}

// GetFileExtension 获取文件扩展名
func GetFileExtension(path string) string {
	return strings.ToLower(filepath.Ext(path))
}

// GetFileName 获取文件名(不含扩展名)
func GetFileName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// GetFilePath 获取文件路径(不含文件名)
func GetFilePath(path string) string {
	return filepath.Dir(path)
}

// GetAbsolutePath 获取绝对路径
func GetAbsolutePath(path string) (string, error) {
	return filepath.Abs(path)
}

// GetRelativePath 获取相对路径
func GetRelativePath(path, basePath string) (string, error) {
	return filepath.Rel(basePath, path)
}

// JoinPaths 连接路径
func JoinPaths(elem ...string) string {
	return filepath.Join(elem...)
}

// CleanPath 清理路径
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// WalkDir 遍历目录
func WalkDir(root string, fn filepath.WalkFunc) error {
	return filepath.Walk(root, fn)
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 复制内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// 同步文件
	return dstFile.Sync()
}

// CopyDir 复制目录
func CopyDir(src, dst string) error {
	// 获取源目录信息
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// 创建目标目录
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// 遍历源目录
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算目标路径
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		// 处理目录
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 复制文件
		return CopyFile(path, dstPath)
	})
}

// ReadFileLines 读取文件所有行
func ReadFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// WriteFileLines 写入文件所有行
func WriteFileLines(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

// ReadFileBytes 读取文件所有字节
func ReadFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFileBytes 写入文件所有字节
func WriteFileBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// GetTempDir 获取临时目录
func GetTempDir() string {
	return os.TempDir()
}

// CreateTempFile 创建临时文件
func CreateTempFile(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

// CreateTempDir 创建临时目录
func CreateTempDir(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}

/*
使用示例:

1. 文件操作:
if utils.FileExists("config.json") {
    data, err := utils.ReadFileBytes("config.json")
    if err != nil {
        log.Fatal(err)
    }
    // 处理文件内容
}

2. 目录操作:
if err := utils.CreateDirIfNotExists("output"); err != nil {
    log.Fatal(err)
}

3. 路径操作:
absPath, err := utils.GetAbsolutePath("../config")
if err != nil {
    log.Fatal(err)
}

4. 文件复制:
if err := utils.CopyFile("src.txt", "dst.txt"); err != nil {
    log.Fatal(err)
}

5. 目录遍历:
err := utils.WalkDir(".", func(path string, info os.FileInfo, err error) error {
    if err != nil {
        return err
    }
    if !info.IsDir() {
        fmt.Printf("Found file: %s\n", path)
    }
    return nil
})
*/ 