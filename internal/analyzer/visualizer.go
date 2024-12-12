package analyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/lkpsg/ccscanner/pkg/models"
	"go.uber.org/zap"
)

// Visualizer 依赖可视化器
type Visualizer struct {
	logger *zap.Logger // 日志记录器
}

// NewVisualizer 创建可视化器
func NewVisualizer(logger *zap.Logger) *Visualizer {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &Visualizer{
		logger: logger,
	}
}

// Format 可视化格式
type Format string

const (
	FormatD3     Format = "d3"     // D3.js格式
	FormatMermaid Format = "mermaid" // Mermaid格式
	FormatGraphviz Format = "graphviz" // Graphviz格式
)

// VisualizeOptions 可视化选项
type VisualizeOptions struct {
	Format       Format // 输出格式
	ShowVersions bool   // 显示版本信息
	ShowLicenses bool   // 显示许可证信息
	Depth        int    // 显示深度 (0表示无限制)
}

// Visualize 生成可视化
func (v *Visualizer) Visualize(graph *DependencyGraph, outputFile string, opts VisualizeOptions) error {
	// 创建输出目录
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// 根据格式生成可视化
	var data []byte
	var err error

	switch opts.Format {
	case FormatD3:
		data, err = v.generateD3(graph, opts)
	case FormatMermaid:
		data, err = v.generateMermaid(graph, opts)
	case FormatGraphviz:
		data, err = v.generateGraphviz(graph, opts)
	default:
		return fmt.Errorf("unsupported format: %s", opts.Format)
	}

	if err != nil {
		return err
	}

	// 写入文件
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}

	return nil
}

// generateD3 生成D3.js格式的可视化
func (v *Visualizer) generateD3(graph *DependencyGraph, opts VisualizeOptions) ([]byte, error) {
	// 准备节点和连接数据
	nodes := make([]map[string]interface{}, 0)
	links := make([]map[string]interface{}, 0)

	// 添加节点
	for _, dep := range graph.GetAllNodes() {
		node := map[string]interface{}{
			"id":   dep.Name,
			"type": dep.Type,
		}
		if opts.ShowVersions {
			node["version"] = dep.Version
		}
		if opts.ShowLicenses && dep.License != "" {
			node["license"] = dep.License
		}
		nodes = append(nodes, node)
	}

	// 添加连接
	edges := graph.GetAllEdges()
	for from, tos := range edges {
		for _, to := range tos {
			link := map[string]interface{}{
				"source": from,
				"target": to,
			}
			links = append(links, link)
		}
	}

	// 生成HTML模板
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>依赖关系图</title>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <style>
        body {
            margin: 0;
            font-family: Arial, sans-serif;
        }
        #graph {
            width: 100vw;
            height: 100vh;
        }
        .node {
            fill: #69b3a2;
            stroke: #fff;
            stroke-width: 2px;
        }
        .link {
            stroke: #999;
            stroke-opacity: 0.6;
            stroke-width: 1px;
        }
        .label {
            font-size: 12px;
            fill: #333;
        }
        .tooltip {
            position: absolute;
            padding: 8px;
            background: #fff;
            border: 1px solid #ddd;
            border-radius: 4px;
            pointer-events: none;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div id="graph"></div>
    <script>
        const data = {{.Data}};
        
        // 创建力导向图
        const width = window.innerWidth;
        const height = window.innerHeight;
        
        const simulation = d3.forceSimulation(data.nodes)
            .force("link", d3.forceLink(data.links).id(d => d.id))
            .force("charge", d3.forceManyBody().strength(-300))
            .force("center", d3.forceCenter(width / 2, height / 2));
        
        const svg = d3.select("#graph")
            .append("svg")
            .attr("width", width)
            .attr("height", height);
        
        // 创建箭头标记
        svg.append("defs").append("marker")
            .attr("id", "arrowhead")
            .attr("viewBox", "-0 -5 10 10")
            .attr("refX", 20)
            .attr("refY", 0)
            .attr("orient", "auto")
            .attr("markerWidth", 6)
            .attr("markerHeight", 6)
            .append("path")
            .attr("d", "M 0,-5 L 10,0 L 0,5")
            .attr("fill", "#999");
        
        // 绘制连接线
        const link = svg.append("g")
            .selectAll("line")
            .data(data.links)
            .join("line")
            .attr("class", "link")
            .attr("marker-end", "url(#arrowhead)");
        
        // 创建节点组
        const node = svg.append("g")
            .selectAll("g")
            .data(data.nodes)
            .join("g")
            .call(d3.drag()
                .on("start", dragstarted)
                .on("drag", dragged)
                .on("end", dragended));
        
        // 绘制节点
        node.append("circle")
            .attr("class", "node")
            .attr("r", 8);
        
        // 添加标签
        node.append("text")
            .attr("class", "label")
            .attr("dx", 12)
            .attr("dy", ".35em")
            .text(d => d.id);
        
        // 创建提示框
        const tooltip = d3.select("body")
            .append("div")
            .attr("class", "tooltip")
            .style("opacity", 0);
        
        // 添加节点交互
        node.on("mouseover", function(event, d) {
            tooltip.transition()
                .duration(200)
                .style("opacity", .9);
            
            let html = d.id;
            if (d.version) html += "<br/>版本: " + d.version;
            if (d.license) html += "<br/>许可证: " + d.license;
            
            tooltip.html(html)
                .style("left", (event.pageX + 10) + "px")
                .style("top", (event.pageY - 10) + "px");
        })
        .on("mouseout", function() {
            tooltip.transition()
                .duration(500)
                .style("opacity", 0);
        });
        
        // 更新力导向图
        simulation.on("tick", () => {
            link
                .attr("x1", d => d.source.x)
                .attr("y1", d => d.source.y)
                .attr("x2", d => d.target.x)
                .attr("y2", d => d.target.y);
            
            node
                .attr("transform", d => `translate(${d.x},${d.y})`);
        });
        
        // 拖拽处理函数
        function dragstarted(event) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            event.subject.fx = event.subject.x;
            event.subject.fy = event.subject.y;
        }
        
        function dragged(event) {
            event.subject.fx = event.x;
            event.subject.fy = event.y;
        }
        
        function dragended(event) {
            if (!event.active) simulation.alphaTarget(0);
            event.subject.fx = null;
            event.subject.fy = null;
        }
    </script>
</body>
</html>
`

	// 准备模板数据
	data := struct {
		Data string
	}{
		Data: string(must(json.Marshal(map[string]interface{}{
			"nodes": nodes,
			"links": links,
		}))),
	}

	// 渲染模板
	t, err := template.New("d3").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render template: %v", err)
	}

	return buf.Bytes(), nil
}

// generateMermaid 生成Mermaid格式的可视化
func (v *Visualizer) generateMermaid(graph *DependencyGraph, opts VisualizeOptions) ([]byte, error) {
	var buf bytes.Buffer

	// 写入图表头
	buf.WriteString("graph TD;\n")

	// 添加节点
	for _, dep := range graph.GetAllNodes() {
		label := dep.Name
		if opts.ShowVersions {
			label += fmt.Sprintf(" v%s", dep.Version)
		}
		if opts.ShowLicenses && dep.License != "" {
			label += fmt.Sprintf(" (%s)", dep.License)
		}
		buf.WriteString(fmt.Sprintf("    %s[\"%s\"];\n", dep.Name, label))
	}

	// 添加连接
	edges := graph.GetAllEdges()
	for from, tos := range edges {
		for _, to := range tos {
			buf.WriteString(fmt.Sprintf("    %s --> %s;\n", from, to))
		}
	}

	return buf.Bytes(), nil
}

// generateGraphviz 生成Graphviz格式的可视化
func (v *Visualizer) generateGraphviz(graph *DependencyGraph, opts VisualizeOptions) ([]byte, error) {
	var buf bytes.Buffer

	// 写入图表头
	buf.WriteString("digraph dependencies {\n")
	buf.WriteString("    node [shape=box, style=rounded];\n")
	buf.WriteString("    rankdir=LR;\n")

	// 添加节点
	for _, dep := range graph.GetAllNodes() {
		label := dep.Name
		if opts.ShowVersions {
			label += fmt.Sprintf("\\nv%s", dep.Version)
		}
		if opts.ShowLicenses && dep.License != "" {
			label += fmt.Sprintf("\\n(%s)", dep.License)
		}
		buf.WriteString(fmt.Sprintf("    \"%s\" [label=\"%s\"];\n", dep.Name, label))
	}

	// 添加连接
	edges := graph.GetAllEdges()
	for from, tos := range edges {
		for _, to := range tos {
			buf.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\";\n", from, to))
		}
	}

	buf.WriteString("}\n")
	return buf.Bytes(), nil
}

// must 辅助函数，用于处理JSON序列化错误
func must(data []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return data
}

/*
使用示例:

1. 创建可视化器:
visualizer := NewVisualizer(logger)

2. 设置可视化选项:
opts := VisualizeOptions{
    Format:       FormatD3,
    ShowVersions: true,
    ShowLicenses: true,
    Depth:        3,
}

3. 生成可视化:
err := visualizer.Visualize(graph, "deps.html", opts)
if err != nil {
    log.Printf("Failed to generate visualization: %v\n", err)
}

4. 使用不同格式:
// 生成Mermaid格式
err = visualizer.Visualize(graph, "deps.mmd", VisualizeOptions{
    Format: FormatMermaid,
})

// 生成Graphviz格式
err = visualizer.Visualize(graph, "deps.dot", VisualizeOptions{
    Format: FormatGraphviz,
})
*/
</rewritten_file> 