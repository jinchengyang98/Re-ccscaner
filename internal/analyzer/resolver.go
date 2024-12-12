package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/lkpsg/ccscanner/pkg/models"
	"go.uber.org/zap"
)

// Resolver 依赖解析器
type Resolver struct {
	logger *zap.Logger // 日志记录器
}

// NewResolver 创建依赖解析器
func NewResolver(logger *zap.Logger) *Resolver {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &Resolver{
		logger: logger,
	}
}

// ResolveResult 解析结果
type ResolveResult struct {
	Dependencies []models.Dependency // 解析后的依赖
	Unresolved  []string           // 未解析的依赖
	Errors      []error            // 解析错误
}

// ResolutionStrategy 解析策略
type ResolutionStrategy int

const (
	StrategyNewest ResolutionStrategy = iota // 使用最新版本
	StrategyOldest                          // 使用最旧版本
	StrategyMinimal                         // 使用满足条件的最小版本
)

// ResolveOptions 解析选项
type ResolveOptions struct {
	Strategy ResolutionStrategy // 解析策略
	Strict   bool              // 严格模式
}

// Resolve 解析依赖
func (r *Resolver) Resolve(deps []models.Dependency, opts ResolveOptions) (*ResolveResult, error) {
	result := &ResolveResult{
		Dependencies: make([]models.Dependency, 0),
		Unresolved:  make([]string, 0),
		Errors:      make([]error, 0),
	}

	// 构建依赖图
	graph := NewDependencyGraph()
	for _, dep := range deps {
		graph.AddNode(dep)
		for _, child := range dep.Dependencies {
			graph.AddEdge(dep.Name, child.Name)
		}
	}

	// 检查循环依赖
	cycles := graph.FindCycles()
	if len(cycles) > 0 && opts.Strict {
		return nil, fmt.Errorf("circular dependencies detected: %v", cycles)
	}

	// 按依赖关系排序
	sorted, err := r.topologicalSort(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to sort dependencies: %v", err)
	}

	// 解析每个依赖
	resolved := make(map[string]models.Dependency)
	for _, dep := range sorted {
		resolvedDep, err := r.resolveDependency(dep, resolved, opts)
		if err != nil {
			if opts.Strict {
				result.Errors = append(result.Errors, err)
			} else {
				result.Unresolved = append(result.Unresolved, dep.Name)
				r.logger.Warn("Failed to resolve dependency",
					zap.String("package", dep.Name),
					zap.Error(err))
			}
			continue
		}
		resolved[dep.Name] = resolvedDep
		result.Dependencies = append(result.Dependencies, resolvedDep)
	}

	return result, nil
}

// resolveDependency 解析单个依赖
func (r *Resolver) resolveDependency(dep models.Dependency, resolved map[string]models.Dependency, opts ResolveOptions) (models.Dependency, error) {
	// 检查是否已解析
	if resolvedDep, ok := resolved[dep.Name]; ok {
		return resolvedDep, nil
	}

	// 解析版本约束
	constraints, err := r.parseVersionConstraints(dep)
	if err != nil {
		return dep, fmt.Errorf("invalid version constraints: %v", err)
	}

	// 获取可用版本
	versions, err := r.getAvailableVersions(dep)
	if err != nil {
		return dep, fmt.Errorf("failed to get available versions: %v", err)
	}

	// 选择合适的版本
	version, err := r.selectVersion(versions, constraints, opts.Strategy)
	if err != nil {
		return dep, fmt.Errorf("failed to select version: %v", err)
	}

	// 更新依赖信息
	dep.Version = version.String()
	return dep, nil
}

// parseVersionConstraints 解析版本约束
func (r *Resolver) parseVersionConstraints(dep models.Dependency) ([]*semver.Constraints, error) {
	if dep.Version == "" {
		return nil, nil
	}

	// 分割多个版本约束
	parts := strings.Split(dep.Version, ",")
	constraints := make([]*semver.Constraints, 0, len(parts))

	for _, part := range parts {
		c, err := semver.NewConstraint(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid version constraint '%s': %v", part, err)
		}
		constraints = append(constraints, c)
	}

	return constraints, nil
}

// getAvailableVersions 获取可用版本
func (r *Resolver) getAvailableVersions(dep models.Dependency) ([]*semver.Version, error) {
	// TODO: 从包管理器或仓库获取可用版本
	// 这里仅作示例，返回一些固定版本
	versions := []string{
		"1.0.0", "1.1.0", "1.2.0",
		"2.0.0", "2.1.0", "2.2.0",
	}

	result := make([]*semver.Version, 0, len(versions))
	for _, v := range versions {
		version, err := semver.NewVersion(v)
		if err != nil {
			r.logger.Warn("Invalid version format",
				zap.String("version", v),
				zap.Error(err))
			continue
		}
		result = append(result, version)
	}

	// 按版本排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].LessThan(result[j])
	})

	return result, nil
}

// selectVersion 选择合适的版本
func (r *Resolver) selectVersion(versions []*semver.Version, constraints []*semver.Constraints, strategy ResolutionStrategy) (*semver.Version, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("no available versions")
	}

	// 过滤满足约束的版本
	var compatible []*semver.Version
	for _, version := range versions {
		if r.checkConstraints(version, constraints) {
			compatible = append(compatible, version)
		}
	}

	if len(compatible) == 0 {
		return nil, fmt.Errorf("no compatible versions found")
	}

	// 根据策略选择版本
	switch strategy {
	case StrategyNewest:
		return compatible[len(compatible)-1], nil
	case StrategyOldest:
		return compatible[0], nil
	case StrategyMinimal:
		return r.findMinimalVersion(compatible), nil
	default:
		return compatible[len(compatible)-1], nil
	}
}

// checkConstraints 检查版本是否满足约束
func (r *Resolver) checkConstraints(version *semver.Version, constraints []*semver.Constraints) bool {
	if len(constraints) == 0 {
		return true
	}

	for _, constraint := range constraints {
		if !constraint.Check(version) {
			return false
		}
	}

	return true
}

// findMinimalVersion 查找满足条件的最小版本
func (r *Resolver) findMinimalVersion(versions []*semver.Version) *semver.Version {
	if len(versions) == 0 {
		return nil
	}

	minimal := versions[0]
	for _, version := range versions[1:] {
		if version.LessThan(minimal) {
			minimal = version
		}
	}

	return minimal
}

// topologicalSort 对依赖进行拓扑排序
func (r *Resolver) topologicalSort(graph *DependencyGraph) ([]models.Dependency, error) {
	var sorted []models.Dependency
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var visit func(string) error
	visit = func(name string) error {
		if temp[name] {
			return fmt.Errorf("circular dependency detected")
		}
		if visited[name] {
			return nil
		}
		temp[name] = true

		for _, dep := range graph.GetEdges(name) {
			if err := visit(dep); err != nil {
				return err
			}
		}

		temp[name] = false
		visited[name] = true
		if node, ok := graph.GetNode(name); ok {
			sorted = append(sorted, node)
		}
		return nil
	}

	nodes := graph.GetAllNodes()
	for _, node := range nodes {
		if !visited[node.Name] {
			if err := visit(node.Name); err != nil {
				return nil, err
			}
		}
	}

	return sorted, nil
}

/*
使用示例:

1. 创建解析器:
resolver := NewResolver(logger)

2. 设置解析选项:
opts := ResolveOptions{
    Strategy: StrategyNewest,
    Strict: true,
}

3. 解析依赖:
result, err := resolver.Resolve(dependencies, opts)
if err != nil {
    log.Printf("Failed to resolve dependencies: %v\n", err)
}

4. 处理结果:
// 检查解析错误
for _, err := range result.Errors {
    log.Printf("Resolution error: %v\n", err)
}

// 检查未解析的依赖
for _, pkg := range result.Unresolved {
    log.Printf("Unresolved package: %s\n", pkg)
}

// 使用解析后的依赖
for _, dep := range result.Dependencies {
    log.Printf("Resolved %s to version %s\n", dep.Name, dep.Version)
}
*/ 