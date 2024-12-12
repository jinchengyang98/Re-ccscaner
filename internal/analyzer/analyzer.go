package analyzer

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/your-org/ccscanner/pkg/models"
)

// DependencyAnalyzer 依赖分析器
type DependencyAnalyzer struct {
	dependencies []*models.Dependency
	graph       *DependencyGraph
	cache       *sync.Map
}

// DependencyGraph 依赖关系图
type DependencyGraph struct {
	nodes map[string]*Node
	edges map[string][]string
	mu    sync.RWMutex
}

// Node 依赖节点
type Node struct {
	Name       string
	Versions   []string
	Type       string
	Source     string
	Scope      string
	Metadata   map[string]interface{}
	Conflicts  []*VersionConflict
	Cycles     [][]string
	Level      int
}

// VersionConflict 版本冲突
type VersionConflict struct {
	Package  string
	Required string
	Current  string
	Source   string
	Path     []string
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	TotalDependencies int
	DirectDependencies int
	IndirectDependencies int
	Cycles [][]string
	Conflicts []*VersionConflict
	MaxDepth int
	Stats map[string]interface{}
}

// NewDependencyAnalyzer 创建新的依赖分析器
func NewDependencyAnalyzer() *DependencyAnalyzer {
	return &DependencyAnalyzer{
		graph: &DependencyGraph{
			nodes: make(map[string]*Node),
			edges: make(map[string][]string),
		},
		cache: &sync.Map{},
	}
}

// Analyze 分析依赖关系
func (a *DependencyAnalyzer) Analyze(deps []*models.Dependency) (*AnalysisResult, error) {
	a.dependencies = deps
	
	// 构建依赖图
	if err := a.buildDependencyGraph(); err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %v", err)
	}

	// 检测循环依赖
	cycles := a.detectCycles()

	// 检测版本冲突
	conflicts := a.detectVersionConflicts()

	// 计算依赖深度
	maxDepth := a.calculateDependencyDepth()

	// 收集统计信息
	stats := a.collectStats()

	return &AnalysisResult{
		TotalDependencies: len(a.dependencies),
		DirectDependencies: countDirectDependencies(a.dependencies),
		IndirectDependencies: len(a.dependencies) - countDirectDependencies(a.dependencies),
		Cycles: cycles,
		Conflicts: conflicts,
		MaxDepth: maxDepth,
		Stats: stats,
	}, nil
}

// buildDependencyGraph 构建依赖关系图
func (a *DependencyAnalyzer) buildDependencyGraph() error {
	for _, dep := range a.dependencies {
		// 添加节点
		a.graph.mu.Lock()
		if _, exists := a.graph.nodes[dep.Name]; !exists {
			a.graph.nodes[dep.Name] = &Node{
				Name:     dep.Name,
				Versions: []string{dep.Version},
				Type:     dep.Type,
				Source:   dep.Source,
				Scope:    dep.Scope,
				Metadata: dep.Metadata,
			}
		} else {
			// 合并版本信息
			versions := a.graph.nodes[dep.Name].Versions
			if !contains(versions, dep.Version) {
				versions = append(versions, dep.Version)
			}
			a.graph.nodes[dep.Name].Versions = versions
		}

		// 添加边
		if dep.Dependencies != nil {
			for _, childDep := range dep.Dependencies {
				a.graph.edges[dep.Name] = append(a.graph.edges[dep.Name], childDep.Name)
			}
		}
		a.graph.mu.Unlock()
	}

	return nil
}

// detectCycles 检测循环依赖
func (a *DependencyAnalyzer) detectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	path := make(map[string]bool)

	var dfs func(node string, currentPath []string) bool
	dfs = func(node string, currentPath []string) bool {
		if path[node] {
			// 找到循环
			cycleStart := -1
			for i, n := range currentPath {
				if n == node {
					cycleStart = i
					break
				}
			}
			if cycleStart != -1 {
				cycle := append(currentPath[cycleStart:], node)
				cycles = append(cycles, cycle)
			}
			return true
		}

		if visited[node] {
			return false
		}

		visited[node] = true
		path[node] = true
		currentPath = append(currentPath, node)

		for _, child := range a.graph.edges[node] {
			if dfs(child, currentPath) {
				return true
			}
		}

		path[node] = false
		return false
	}

	for node := range a.graph.nodes {
		if !visited[node] {
			dfs(node, nil)
		}
	}

	return cycles
}

// detectVersionConflicts 检测版本冲突
func (a *DependencyAnalyzer) detectVersionConflicts() []*VersionConflict {
	var conflicts []*VersionConflict
	versionMap := make(map[string]map[string][]string) // package -> version -> [required by]

	// 收集所有版本要求
	for _, dep := range a.dependencies {
		if _, exists := versionMap[dep.Name]; !exists {
			versionMap[dep.Name] = make(map[string][]string)
		}
		versionMap[dep.Name][dep.Version] = append(versionMap[dep.Name][dep.Version], "root")

		if dep.Dependencies != nil {
			for _, childDep := range dep.Dependencies {
				if _, exists := versionMap[childDep.Name]; !exists {
					versionMap[childDep.Name] = make(map[string][]string)
				}
				versionMap[childDep.Name][childDep.Version] = append(
					versionMap[childDep.Name][childDep.Version],
					dep.Name,
				)
			}
		}
	}

	// 检查每个包的版本冲突
	for pkg, versions := range versionMap {
		if len(versions) > 1 {
			// 尝试解析所有版本
			var semvers []*semver.Version
			for v := range versions {
				if sv, err := semver.NewVersion(normalizeVersion(v)); err == nil {
					semvers = append(semvers, sv)
				}
			}

			// 如果有多个语义化版本，检查是否冲突
			if len(semvers) > 1 {
				sort.Slice(semvers, func(i, j int) bool {
					return semvers[i].LessThan(semvers[j])
				})

				// 如果最高版本和最低版本不同，记录冲突
				if !semvers[0].Equal(semvers[len(semvers)-1]) {
					for v, sources := range versions {
						conflicts = append(conflicts, &VersionConflict{
							Package:  pkg,
							Required: v,
							Current:  semvers[len(semvers)-1].String(),
							Source:   strings.Join(sources, ", "),
							Path:     a.findDependencyPath(pkg),
						})
					}
				}
			}
		}
	}

	return conflicts
}

// calculateDependencyDepth 计算依赖深度
func (a *DependencyAnalyzer) calculateDependencyDepth() int {
	maxDepth := 0
	visited := make(map[string]bool)

	var dfs func(node string, depth int)
	dfs = func(node string, depth int) {
		if visited[node] {
			return
		}
		visited[node] = true

		if depth > maxDepth {
			maxDepth = depth
		}

		for _, child := range a.graph.edges[node] {
			dfs(child, depth+1)
		}
	}

	for node := range a.graph.nodes {
		if !visited[node] {
			dfs(node, 0)
		}
	}

	return maxDepth
}

// collectStats 收集统计信息
func (a *DependencyAnalyzer) collectStats() map[string]interface{} {
	stats := map[string]interface{}{
		"by_type":   make(map[string]int),
		"by_scope":  make(map[string]int),
		"by_source": make(map[string]int),
	}

	for _, dep := range a.dependencies {
		stats["by_type"].(map[string]int)[dep.Type]++
		stats["by_scope"].(map[string]int)[dep.Scope]++
		stats["by_source"].(map[string]int)[dep.Source]++
	}

	return stats
}

// findDependencyPath 查找依赖路径
func (a *DependencyAnalyzer) findDependencyPath(target string) []string {
	visited := make(map[string]bool)
	var path []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		if node == target {
			path = append(path, node)
			return true
		}

		if visited[node] {
			return false
		}

		visited[node] = true
		path = append(path, node)

		for _, child := range a.graph.edges[node] {
			if dfs(child) {
				return true
			}
		}

		path = path[:len(path)-1]
		return false
	}

	for node := range a.graph.nodes {
		if !visited[node] {
			if dfs(node) {
				return path
			}
		}
	}

	return nil
}

// normalizeVersion 标准化版本号
func normalizeVersion(version string) string {
	// 处理特殊版本格式
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}
	if strings.HasPrefix(version, "branch=") || strings.HasPrefix(version, "commit=") {
		return "0.0.0"
	}
	if strings.HasPrefix(version, ">=") {
		version = version[2:]
	}
	if strings.Contains(version, "...") {
		parts := strings.Split(version, "...")
		version = parts[0]
	}
	
	// 确保版本号有三个部分
	parts := strings.Split(version, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	
	return strings.Join(parts, ".")
}

// countDirectDependencies 计算直接依赖数量
func countDirectDependencies(deps []*models.Dependency) int {
	count := 0
	for _, dep := range deps {
		if dep.Scope != "indirect" {
			count++
		}
	}
	return count
}

// contains 检查切片是否包含指定值
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetDependencyTree 获取依赖树
func (a *DependencyAnalyzer) GetDependencyTree() map[string]interface{} {
	tree := make(map[string]interface{})
	visited := make(map[string]bool)

	var buildTree func(node string) map[string]interface{}
	buildTree = func(node string) map[string]interface{} {
		if visited[node] {
			return map[string]interface{}{
				"name":    node,
				"cyclic":  true,
				"version": a.graph.nodes[node].Versions[0],
			}
		}

		visited[node] = true
		nodeInfo := map[string]interface{}{
			"name":     node,
			"versions": a.graph.nodes[node].Versions,
			"type":     a.graph.nodes[node].Type,
			"source":   a.graph.nodes[node].Source,
			"scope":    a.graph.nodes[node].Scope,
		}

		if children := a.graph.edges[node]; len(children) > 0 {
			deps := make(map[string]interface{})
			for _, child := range children {
				deps[child] = buildTree(child)
			}
			nodeInfo["dependencies"] = deps
		}

		return nodeInfo
	}

	// 构建每个根节点的树
	for node := range a.graph.nodes {
		if !visited[node] {
			tree[node] = buildTree(node)
		}
	}

	return tree
}

// 注意事项:
// 1. 使用图算法检测循环依赖
// 2. 使用语义化版本比较检测冲突
// 3. 提供丰富的统计信息
// 4. 支持依赖树可视化
// 5. 处理特殊版本格式