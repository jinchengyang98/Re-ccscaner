package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/ccscanner/pkg/models"
)

func TestDependencyAnalyzer_Analyze(t *testing.T) {
	// 创建测试依赖
	deps := []*models.Dependency{
		{
			Name:    "A",
			Version: "1.0.0",
			Type:    "direct",
			Dependencies: []*models.Dependency{
				{
					Name:    "B",
					Version: "2.0.0",
					Type:    "indirect",
				},
				{
					Name:    "C",
					Version: "1.0.0",
					Type:    "indirect",
				},
			},
		},
		{
			Name:    "B",
			Version: "2.0.0",
			Type:    "indirect",
			Dependencies: []*models.Dependency{
				{
					Name:    "D",
					Version: "1.0.0",
					Type:    "indirect",
				},
			},
		},
		{
			Name:    "C",
			Version: "1.0.0",
			Type:    "indirect",
			Dependencies: []*models.Dependency{
				{
					Name:    "D",
					Version: "2.0.0",
					Type:    "indirect",
				},
			},
		},
		{
			Name:    "D",
			Version: "1.0.0",
			Type:    "indirect",
		},
		{
			Name:    "D",
			Version: "2.0.0",
			Type:    "indirect",
		},
	}

	// 创建分析器
	analyzer := NewDependencyAnalyzer()

	// 执行分析
	result, err := analyzer.Analyze(deps)
	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证基本信息
	assert.Equal(t, 5, result.TotalDependencies)
	assert.Equal(t, 1, result.DirectDependencies)
	assert.Equal(t, 4, result.IndirectDependencies)

	// 验证版本冲突
	assert.Len(t, result.Conflicts, 1)
	conflict := result.Conflicts[0]
	assert.Equal(t, "D", conflict.Package)
	assert.Contains(t, []string{"1.0.0", "2.0.0"}, conflict.Required)
	assert.Contains(t, []string{"1.0.0", "2.0.0"}, conflict.Current)

	// 验证依赖深度
	assert.Equal(t, 3, result.MaxDepth)

	// 验证统计信息
	assert.NotNil(t, result.Stats)
	assert.Contains(t, result.Stats, "by_type")
	assert.Contains(t, result.Stats, "by_scope")
}

func TestDependencyAnalyzer_DetectCycles(t *testing.T) {
	// 创建带有循环依赖的测试数据
	deps := []*models.Dependency{
		{
			Name:    "A",
			Version: "1.0.0",
			Dependencies: []*models.Dependency{
				{
					Name:    "B",
					Version: "1.0.0",
				},
			},
		},
		{
			Name:    "B",
			Version: "1.0.0",
			Dependencies: []*models.Dependency{
				{
					Name:    "C",
					Version: "1.0.0",
				},
			},
		},
		{
			Name:    "C",
			Version: "1.0.0",
			Dependencies: []*models.Dependency{
				{
					Name:    "A",
					Version: "1.0.0",
				},
			},
		},
	}

	analyzer := NewDependencyAnalyzer()
	result, err := analyzer.Analyze(deps)
	require.NoError(t, err)

	// 验证循环依赖检测
	assert.NotEmpty(t, result.Cycles)
	assert.Contains(t, result.Cycles[0], "A")
	assert.Contains(t, result.Cycles[0], "B")
	assert.Contains(t, result.Cycles[0], "C")
}

func TestDependencyAnalyzer_GetDependencyTree(t *testing.T) {
	// 创建测试依赖
	deps := []*models.Dependency{
		{
			Name:    "root",
			Version: "1.0.0",
			Dependencies: []*models.Dependency{
				{
					Name:    "child1",
					Version: "1.0.0",
				},
				{
					Name:    "child2",
					Version: "1.0.0",
				},
			},
		},
		{
			Name:    "child1",
			Version: "1.0.0",
		},
		{
			Name:    "child2",
			Version: "1.0.0",
		},
	}

	analyzer := NewDependencyAnalyzer()
	analyzer.dependencies = deps
	require.NoError(t, analyzer.buildDependencyGraph())

	// 获取依赖树
	tree := analyzer.GetDependencyTree()

	// 验证树结构
	assert.Contains(t, tree, "root")
	root := tree["root"].(map[string]interface{})
	assert.Equal(t, "root", root["name"])
	assert.Equal(t, "1.0.0", root["versions"].([]string)[0])

	deps, ok := root["dependencies"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, deps, "child1")
	assert.Contains(t, deps, "child2")
}

func TestDependencyAnalyzer_VersionNormalization(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "standard version",
			version:  "1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "version with v prefix",
			version:  "v1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "partial version",
			version:  "1.2",
			expected: "1.2.0",
		},
		{
			name:     "branch version",
			version:  "branch=main",
			expected: "0.0.0",
		},
		{
			name:     "commit version",
			version:  "commit=abc123",
			expected: "0.0.0",
		},
		{
			name:     "range version",
			version:  "1.0.0...2.0.0",
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDependencyAnalyzer_EmptyDependencies(t *testing.T) {
	analyzer := NewDependencyAnalyzer()
	result, err := analyzer.Analyze([]*models.Dependency{})
	require.NoError(t, err)

	assert.Equal(t, 0, result.TotalDependencies)
	assert.Equal(t, 0, result.DirectDependencies)
	assert.Equal(t, 0, result.IndirectDependencies)
	assert.Empty(t, result.Cycles)
	assert.Empty(t, result.Conflicts)
	assert.Equal(t, 0, result.MaxDepth)
}

func TestDependencyAnalyzer_ComplexGraph(t *testing.T) {
	// 创建复杂的依赖图
	deps := []*models.Dependency{
		{
			Name:    "app",
			Version: "1.0.0",
			Type:    "direct",
			Dependencies: []*models.Dependency{
				{Name: "lib1", Version: "1.0.0"},
				{Name: "lib2", Version: "2.0.0"},
			},
		},
		{
			Name:    "lib1",
			Version: "1.0.0",
			Type:    "indirect",
			Dependencies: []*models.Dependency{
				{Name: "lib3", Version: "1.0.0"},
				{Name: "lib4", Version: "1.0.0"},
			},
		},
		{
			Name:    "lib2",
			Version: "2.0.0",
			Type:    "indirect",
			Dependencies: []*models.Dependency{
				{Name: "lib4", Version: "2.0.0"},
				{Name: "lib5", Version: "1.0.0"},
			},
		},
		{
			Name:    "lib3",
			Version: "1.0.0",
			Type:    "indirect",
		},
		{
			Name:    "lib4",
			Version: "1.0.0",
			Type:    "indirect",
		},
		{
			Name:    "lib4",
			Version: "2.0.0",
			Type:    "indirect",
		},
		{
			Name:    "lib5",
			Version: "1.0.0",
			Type:    "indirect",
			Dependencies: []*models.Dependency{
				{Name: "lib6", Version: "1.0.0"},
			},
		},
		{
			Name:    "lib6",
			Version: "1.0.0",
			Type:    "indirect",
		},
	}

	analyzer := NewDependencyAnalyzer()
	result, err := analyzer.Analyze(deps)
	require.NoError(t, err)

	// 验证依赖统计
	assert.Equal(t, 8, result.TotalDependencies)
	assert.Equal(t, 1, result.DirectDependencies)
	assert.Equal(t, 7, result.IndirectDependencies)

	// 验证最大深度
	assert.Equal(t, 4, result.MaxDepth)

	// 验证版本冲突
	assert.NotEmpty(t, result.Conflicts)
	hasLib4Conflict := false
	for _, conflict := range result.Conflicts {
		if conflict.Package == "lib4" {
			hasLib4Conflict = true
			break
		}
	}
	assert.True(t, hasLib4Conflict)

	// 验证统计信息
	assert.NotNil(t, result.Stats["by_type"])
	typeStats := result.Stats["by_type"].(map[string]int)
	assert.Equal(t, 1, typeStats["direct"])
	assert.Equal(t, 7, typeStats["indirect"])
}

// 注意事项:
// 1. 测试用例覆盖主要功能和边缘情况
// 2. 验证版本冲突检测
// 3. 验证循环依赖检测
// 4. 验证依赖树生成
// 5. 验证版本号标准化 