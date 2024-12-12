// 全局变量
let network = null; // vis.js网络图实例
let nodes = null; // 节点数据集
let edges = null; // 边数据集
let dependencyTree = null; // 依赖树实例
let scanResult = null; // 扫描结果

// 初始化函数
document.addEventListener('DOMContentLoaded', () => {
    // 获取扫描ID
    const urlParams = new URLSearchParams(window.location.search);
    const scanId = urlParams.get('id');
    
    if (scanId) {
        // 加载扫描结果
        loadScanResult(scanId);
    } else {
        showNotification('未指定扫描ID', 'error');
    }
    
    // 初始化事件处理
    initEventHandlers();
});

// 加载扫描结果
async function loadScanResult(scanId) {
    try {
        const response = await fetch(`/api/scan/${scanId}/results`);
        if (!response.ok) throw new Error('获取扫描结果失败');
        
        scanResult = await response.json();
        
        // 初始化依赖图
        initDependencyGraph();
        
        // 初始化依赖树
        initDependencyTree();
        
        // 更新统计信息
        updateStatistics();
        
    } catch (error) {
        console.error('加载扫描结果失败:', error);
        showNotification('加载扫描结果失败', 'error');
    }
}

// 初始化依赖图
function initDependencyGraph() {
    // 创建数据集
    nodes = new vis.DataSet();
    edges = new vis.DataSet();
    
    // 添加节点和边
    addNodesToGraph();
    addEdgesToGraph();
    
    // 创建网络图
    const container = document.getElementById('dependencyGraph');
    const data = { nodes, edges };
    const options = {
        nodes: {
            shape: 'dot',
            size: 16,
            font: {
                size: 12
            },
            borderWidth: 2,
            shadow: true
        },
        edges: {
            width: 1,
            smooth: {
                type: 'continuous'
            },
            arrows: {
                to: {
                    enabled: true,
                    scaleFactor: 0.5
                }
            }
        },
        physics: {
            stabilization: false,
            barnesHut: {
                gravitationalConstant: -80000,
                springConstant: 0.001,
                springLength: 200
            }
        },
        interaction: {
            navigationButtons: true,
            keyboard: true
        }
    };
    
    network = new vis.Network(container, data, options);
    
    // 注册事件处理
    network.on('selectNode', (params) => {
        if (params.nodes.length > 0) {
            const nodeId = params.nodes[0];
            highlightConnectedNodes(nodeId);
            showNodeDetails(nodeId);
        }
    });
    
    network.on('deselectNode', () => {
        clearHighlight();
        hideNodeDetails();
    });
}

// 添加节点到图
function addNodesToGraph() {
    scanResult.dependencies.forEach(dep => {
        nodes.add({
            id: dep.name,
            label: `${dep.name}\n${dep.version}`,
            title: generateNodeTooltip(dep),
            color: getNodeColor(dep),
            size: getNodeSize(dep)
        });
    });
}

// 添加边到图
function addEdgesToGraph() {
    scanResult.dependencies.forEach(dep => {
        if (dep.dependencies) {
            dep.dependencies.forEach(child => {
                edges.add({
                    from: dep.name,
                    to: child.name,
                    color: getEdgeColor(dep, child)
                });
            });
        }
    });
}

// 初始化依赖树
function initDependencyTree() {
    const container = document.getElementById('dependencyTree');
    container.innerHTML = '';
    
    // 构建树结构
    const rootNode = document.createElement('ul');
    rootNode.className = 'list-unstyled';
    
    // 添加直接依赖
    const directDeps = scanResult.dependencies.filter(dep => dep.type === 'direct');
    directDeps.forEach(dep => {
        const node = createTreeNode(dep);
        rootNode.appendChild(node);
    });
    
    container.appendChild(rootNode);
}

// 创建树节点
function createTreeNode(dep, level = 0) {
    const li = document.createElement('li');
    li.className = 'tree-item';
    li.style.paddingLeft = `${level * 20}px`;
    
    const content = document.createElement('div');
    content.className = 'd-flex align-items-center';
    content.innerHTML = `
        <i class="fas fa-${dep.dependencies?.length ? 'caret-right' : 'circle'} me-2"></i>
        <span class="flex-grow-1">${dep.name} (${dep.version})</span>
        <span class="badge ${getTypeBadgeClass(dep.type)} ms-2">${dep.type}</span>
    `;
    
    // 添加点击事件
    content.addEventListener('click', () => {
        // 展开/折叠子节点
        const childList = li.querySelector('ul');
        if (childList) {
            childList.style.display = childList.style.display === 'none' ? 'block' : 'none';
            content.querySelector('i').className = `fas fa-${childList.style.display === 'none' ? 'caret-right' : 'caret-down'} me-2`;
        }
        
        // 高亮依赖图中的节点
        network.selectNodes([dep.name]);
    });
    
    li.appendChild(content);
    
    // 添加子节点
    if (dep.dependencies?.length) {
        const ul = document.createElement('ul');
        ul.className = 'list-unstyled';
        ul.style.display = 'none';
        
        dep.dependencies.forEach(child => {
            const childNode = createTreeNode(child, level + 1);
            ul.appendChild(childNode);
        });
        
        li.appendChild(ul);
    }
    
    return li;
}

// 更新统计信息
function updateStatistics() {
    document.getElementById('totalDepsCount').textContent = scanResult.dependencies.length;
    document.getElementById('directDepsCount').textContent = scanResult.analysis.directDependencies;
    document.getElementById('indirectDepsCount').textContent = scanResult.analysis.indirectDependencies;
    document.getElementById('maxDepth').textContent = scanResult.analysis.maxDepth;
    document.getElementById('conflictsCount').textContent = scanResult.analysis.conflicts.length;
    document.getElementById('cyclesCount').textContent = scanResult.analysis.cycles.length;
    document.getElementById('outdatedCount').textContent = countOutdatedDeps();
    document.getElementById('licenseIssuesCount').textContent = countLicenseIssues();
}

// 初始化事件处理
function initEventHandlers() {
    // 搜索框
    document.getElementById('searchDeps').addEventListener('input', (e) => {
        const query = e.target.value.toLowerCase();
        filterNodes(query);
    });
    
    // 依赖类型过滤
    document.getElementById('showDirect').addEventListener('change', updateFilters);
    document.getElementById('showIndirect').addEventListener('change', updateFilters);
    
    // 深度控制
    document.getElementById('depthRange').addEventListener('input', (e) => {
        const depth = parseInt(e.target.value);
        document.getElementById('depthValue').textContent = depth;
        filterByDepth(depth);
    });
    
    // 布局类型
    document.getElementById('layoutType').addEventListener('change', (e) => {
        updateLayout(e.target.value);
    });
    
    // 工具栏按钮
    document.getElementById('zoomIn').addEventListener('click', () => network.zoom(1.2));
    document.getElementById('zoomOut').addEventListener('click', () => network.zoom(0.8));
    document.getElementById('fitGraph').addEventListener('click', () => network.fit());
    document.getElementById('exportImage').addEventListener('click', exportImage);
    document.getElementById('exportDOT').addEventListener('click', exportDOT);
    document.getElementById('analyzeGraph').addEventListener('click', analyzeGraph);
    document.getElementById('optimizeGraph').addEventListener('click', optimizeGraph);
}

// 过滤节点
function filterNodes(query) {
    const filteredNodeIds = [];
    nodes.forEach(node => {
        if (node.label.toLowerCase().includes(query)) {
            filteredNodeIds.push(node.id);
        }
    });
    
    nodes.forEach(node => {
        const isVisible = filteredNodeIds.includes(node.id);
        nodes.update({ id: node.id, hidden: !isVisible });
    });
    
    edges.forEach(edge => {
        const isVisible = filteredNodeIds.includes(edge.from) && filteredNodeIds.includes(edge.to);
        edges.update({ id: edge.id, hidden: !isVisible });
    });
}

// 更新过滤器
function updateFilters() {
    const showDirect = document.getElementById('showDirect').checked;
    const showIndirect = document.getElementById('showIndirect').checked;
    
    nodes.forEach(node => {
        const dep = scanResult.dependencies.find(d => d.name === node.id);
        const isVisible = (dep.type === 'direct' && showDirect) || (dep.type === 'indirect' && showIndirect);
        nodes.update({ id: node.id, hidden: !isVisible });
    });
    
    updateEdgesVisibility();
}

// 按深度过滤
function filterByDepth(maxDepth) {
    const visibleNodes = new Set();
    
    // 添加直接依赖
    scanResult.dependencies.filter(dep => dep.type === 'direct').forEach(dep => {
        visibleNodes.add(dep.name);
        addChildrenToDepth(dep, 1, maxDepth, visibleNodes);
    });
    
    nodes.forEach(node => {
        const isVisible = visibleNodes.has(node.id);
        nodes.update({ id: node.id, hidden: !isVisible });
    });
    
    updateEdgesVisibility();
}

// 递归添加子节点
function addChildrenToDepth(dep, currentDepth, maxDepth, visibleNodes) {
    if (currentDepth >= maxDepth || !dep.dependencies) return;
    
    dep.dependencies.forEach(child => {
        visibleNodes.add(child.name);
        addChildrenToDepth(child, currentDepth + 1, maxDepth, visibleNodes);
    });
}

// 更新边的可见性
function updateEdgesVisibility() {
    edges.forEach(edge => {
        const fromNode = nodes.get(edge.from);
        const toNode = nodes.get(edge.to);
        const isVisible = !fromNode.hidden && !toNode.hidden;
        edges.update({ id: edge.id, hidden: !isVisible });
    });
}

// 更新布局
function updateLayout(type) {
    const options = {
        physics: {
            enabled: true
        }
    };
    
    switch (type) {
        case 'hierarchical':
            options.layout = {
                hierarchical: {
                    direction: 'UD',
                    sortMethod: 'directed',
                    nodeSpacing: 150,
                    levelSeparation: 150
                }
            };
            break;
        case 'force':
            options.layout = {
                hierarchical: false
            };
            break;
        case 'circular':
            options.layout = {
                improvedLayout: true,
                randomSeed: 42
            };
            break;
    }
    
    network.setOptions(options);
}

// 导出图片
function exportImage() {
    const canvas = network.canvas.frame.canvas;
    const dataUrl = canvas.toDataURL('image/png');
    
    const a = document.createElement('a');
    a.href = dataUrl;
    a.download = 'dependency-graph.png';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
}

// 导出DOT格式
function exportDOT() {
    let dot = 'digraph DependencyGraph {\n';
    
    // 添加节点
    nodes.forEach(node => {
        if (!node.hidden) {
            dot += `  "${node.id}" [label="${node.label}"];\n`;
        }
    });
    
    // 添加边
    edges.forEach(edge => {
        if (!edge.hidden) {
            dot += `  "${edge.from}" -> "${edge.to}";\n`;
        }
    });
    
    dot += '}';
    
    const blob = new Blob([dot], { type: 'text/plain' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'dependency-graph.dot';
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
}

// 分析图
function analyzeGraph() {
    const analysis = {
        totalNodes: nodes.length,
        totalEdges: edges.length,
        avgDegree: edges.length / nodes.length,
        maxInDegree: 0,
        maxOutDegree: 0,
        isolatedNodes: 0,
        density: (2 * edges.length) / (nodes.length * (nodes.length - 1))
    };
    
    // TODO: 显示分析结果
    console.log(analysis);
}

// 优化图
function optimizeGraph() {
    network.stabilize(100);
}

// 高亮相关节点
function highlightConnectedNodes(nodeId) {
    const connectedNodes = network.getConnectedNodes(nodeId);
    const allNodes = nodes.get();
    const updates = [];
    
    allNodes.forEach(node => {
        if (node.id === nodeId || connectedNodes.includes(node.id)) {
            updates.push({ id: node.id, color: { opacity: 1 } });
        } else {
            updates.push({ id: node.id, color: { opacity: 0.2 } });
        }
    });
    
    nodes.update(updates);
    
    const allEdges = edges.get();
    const edgeUpdates = [];
    
    allEdges.forEach(edge => {
        if (edge.from === nodeId || edge.to === nodeId) {
            edgeUpdates.push({ id: edge.id, color: { opacity: 1 } });
        } else {
            edgeUpdates.push({ id: edge.id, color: { opacity: 0.2 } });
        }
    });
    
    edges.update(edgeUpdates);
}

// 清除高亮
function clearHighlight() {
    const allNodes = nodes.get();
    const updates = allNodes.map(node => ({
        id: node.id,
        color: getNodeColor(scanResult.dependencies.find(d => d.name === node.id))
    }));
    
    nodes.update(updates);
    
    const allEdges = edges.get();
    const edgeUpdates = allEdges.map(edge => ({
        id: edge.id,
        color: getEdgeColor(
            scanResult.dependencies.find(d => d.name === edge.from),
            scanResult.dependencies.find(d => d.name === edge.to)
        )
    }));
    
    edges.update(edgeUpdates);
}

// 显示节点详情
function showNodeDetails(nodeId) {
    const dep = scanResult.dependencies.find(d => d.name === nodeId);
    if (!dep) return;
    
    // TODO: 显示详细信息面板
}

// 隐藏节点详情
function hideNodeDetails() {
    // TODO: 隐藏详细信息面板
}

// 生成节点提示
function generateNodeTooltip(dep) {
    return `
        <div class="node-tooltip">
            <h6>${dep.name}</h6>
            <p>版本: ${dep.version}</p>
            <p>类型: ${dep.type}</p>
            <p>来源: ${dep.source}</p>
            ${dep.license ? `<p>许可证: ${dep.license}</p>` : ''}
        </div>
    `;
}

// 获取节点颜色
function getNodeColor(dep) {
    if (hasVulnerabilities(dep)) {
        return { background: '#dc3545', border: '#dc3545' };
    }
    if (hasConflicts(dep)) {
        return { background: '#ffc107', border: '#ffc107' };
    }
    if (dep.type === 'direct') {
        return { background: '#007bff', border: '#0056b3' };
    }
    return { background: '#6c757d', border: '#495057' };
}

// 获取边颜色
function getEdgeColor(fromDep, toDep) {
    if (hasConflicts(toDep)) {
        return { color: '#ffc107', opacity: 0.8 };
    }
    if (fromDep.type === 'direct') {
        return { color: '#007bff', opacity: 0.8 };
    }
    return { color: '#6c757d', opacity: 0.6 };
}

// 获取节点大小
function getNodeSize(dep) {
    if (dep.type === 'direct') return 20;
    if (dep.dependencies?.length > 5) return 18;
    return 16;
}

// 获取类型徽章样式
function getTypeBadgeClass(type) {
    switch (type) {
        case 'direct':
            return 'bg-primary';
        case 'indirect':
            return 'bg-secondary';
        default:
            return 'bg-light text-dark';
    }
}

// 检查是否有漏洞
function hasVulnerabilities(dep) {
    return scanResult.vulnerabilities.some(v => v.package === dep.name);
}

// 检查是否有冲突
function hasConflicts(dep) {
    return scanResult.analysis.conflicts.some(c => c.package === dep.name);
}

// 统计过时依赖数量
function countOutdatedDeps() {
    // TODO: 实现过时依赖检测
    return 0;
}

// 统计许可证问题数量
function countLicenseIssues() {
    // TODO: 实现许可证兼容性检查
    return 0;
}

// 显示通知
function showNotification(message, level = 'info') {
    // 创建通知元素
    const notification = document.createElement('div');
    notification.className = `alert alert-${level} alert-dismissible fade show position-fixed bottom-0 end-0 m-3`;
    notification.style.zIndex = '9999';
    notification.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
    `;
    
    // 添加到页面
    document.body.appendChild(notification);
    
    // 3秒后自动关闭
    setTimeout(() => {
        notification.remove();
    }, 3000);
}

// 注意事项:
// 1. 使用vis.js绘制依赖图
// 2. 实现交互式过滤和搜索
// 3. 支持多种布局方式
// 4. 提供图形导出功能
// 5. 显示详细的依赖信息
</rewritten_file> 