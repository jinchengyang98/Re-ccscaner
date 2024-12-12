// 全局变量
let scanResult = null; // 扫描结果
let depTypeChart = null; // 依赖类型图表
let vulnSeverityChart = null; // 漏洞严重程度图表

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
    
    // 初始化图表
    initCharts();
    
    // 初始化事件处理
    initEventHandlers();
});

// 加载扫描结果
async function loadScanResult(scanId) {
    try {
        const response = await fetch(`/api/scan/${scanId}/results`);
        if (!response.ok) throw new Error('获取扫描结果失败');
        
        scanResult = await response.json();
        
        // 更新页面显示
        updateBasicInfo();
        updateStatistics();
        updateCharts();
        updateTables();
        
    } catch (error) {
        console.error('加载扫描结果失败:', error);
        showNotification('加载扫描结果失败', 'error');
    }
}

// 初始化图表
function initCharts() {
    // 依赖类型分布图表
    const depTypeCtx = document.getElementById('depTypeChart').getContext('2d');
    depTypeChart = new Chart(depTypeCtx, {
        type: 'doughnut',
        data: {
            labels: [],
            datasets: [{
                data: [],
                backgroundColor: [
                    '#007bff',
                    '#6c757d',
                    '#28a745',
                    '#dc3545',
                    '#ffc107',
                    '#17a2b8'
                ]
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'right'
                }
            }
        }
    });
    
    // 漏洞严重程度分布图表
    const vulnSeverityCtx = document.getElementById('vulnSeverityChart').getContext('2d');
    vulnSeverityChart = new Chart(vulnSeverityCtx, {
        type: 'pie',
        data: {
            labels: [],
            datasets: [{
                data: [],
                backgroundColor: [
                    '#dc3545',
                    '#ffc107',
                    '#17a2b8'
                ]
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'right'
                }
            }
        }
    });
}

// 更新基本信息
function updateBasicInfo() {
    document.getElementById('scanId').textContent = scanResult.id;
    document.getElementById('projectPath').textContent = scanResult.path;
    document.getElementById('startTime').textContent = formatDate(scanResult.startTime);
    document.getElementById('endTime').textContent = formatDate(scanResult.endTime);
    document.getElementById('duration').textContent = scanResult.duration;
    document.getElementById('status').textContent = scanResult.status;
    document.getElementById('status').className = `badge bg-${getStatusColor(scanResult.status)}`;
}

// 更新统计信息
function updateStatistics() {
    document.getElementById('totalDeps').textContent = scanResult.dependencies.length;
    document.getElementById('directDeps').textContent = scanResult.analysis.directDependencies;
    document.getElementById('totalVulns').textContent = scanResult.vulnerabilities.length;
    document.getElementById('totalConflicts').textContent = scanResult.analysis.conflicts.length;
}

// 更新图表
function updateCharts() {
    // 更新依赖类型分布
    const depTypes = {};
    scanResult.dependencies.forEach(dep => {
        depTypes[dep.type] = (depTypes[dep.type] || 0) + 1;
    });
    
    depTypeChart.data.labels = Object.keys(depTypes);
    depTypeChart.data.datasets[0].data = Object.values(depTypes);
    depTypeChart.update();
    
    // 更新漏洞严重程度分布
    const vulnSeverity = {
        high: 0,
        medium: 0,
        low: 0
    };
    
    scanResult.vulnerabilities.forEach(vuln => {
        vulnSeverity[vuln.severity.toLowerCase()]++;
    });
    
    vulnSeverityChart.data.labels = ['高危', '中危', '低危'];
    vulnSeverityChart.data.datasets[0].data = [
        vulnSeverity.high,
        vulnSeverity.medium,
        vulnSeverity.low
    ];
    vulnSeverityChart.update();
}

// 更新表格
function updateTables() {
    // 更新依赖表格
    const depsTable = document.getElementById('depsTable').getElementsByTagName('tbody')[0];
    depsTable.innerHTML = '';
    
    scanResult.dependencies.forEach(dep => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${dep.name}</td>
            <td>${dep.version}</td>
            <td>${dep.type}</td>
            <td>${dep.source}</td>
            <td>${dep.license || '-'}</td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" onclick="viewDependencyDetails('${dep.name}')">
                        <i class="fas fa-info-circle"></i>
                    </button>
                    <button class="btn btn-outline-warning" onclick="checkUpdates('${dep.name}')">
                        <i class="fas fa-sync"></i>
                    </button>
                </div>
            </td>
        `;
        depsTable.appendChild(tr);
    });
    
    // 更新漏洞表格
    const vulnsTable = document.getElementById('vulnsTable').getElementsByTagName('tbody')[0];
    vulnsTable.innerHTML = '';
    
    scanResult.vulnerabilities.forEach(vuln => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${vuln.package}</td>
            <td>${vuln.id}</td>
            <td>
                <span class="badge bg-${getSeverityColor(vuln.severity)}">
                    ${vuln.severity}
                </span>
            </td>
            <td>${vuln.description}</td>
            <td>${vuln.fixedVersion || '无'}</td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" onclick="viewVulnerabilityDetails('${vuln.id}')">
                        <i class="fas fa-info-circle"></i>
                    </button>
                    <button class="btn btn-outline-success" onclick="fixVulnerability('${vuln.id}')">
                        <i class="fas fa-wrench"></i>
                    </button>
                </div>
            </td>
        `;
        vulnsTable.appendChild(tr);
    });
    
    // 更新冲突表格
    const conflictsTable = document.getElementById('conflictsTable').getElementsByTagName('tbody')[0];
    conflictsTable.innerHTML = '';
    
    scanResult.analysis.conflicts.forEach(conflict => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${conflict.package}</td>
            <td>${conflict.current}</td>
            <td>${conflict.required}</td>
            <td>${conflict.source}</td>
            <td>升级到 ${getRecommendedVersion(conflict)}</td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" onclick="viewConflictDetails('${conflict.package}')">
                        <i class="fas fa-info-circle"></i>
                    </button>
                    <button class="btn btn-outline-success" onclick="resolveConflict('${conflict.package}')">
                        <i class="fas fa-check"></i>
                    </button>
                </div>
            </td>
        `;
        conflictsTable.appendChild(tr);
    });
}

// 初始化事件处理
function initEventHandlers() {
    // 导出报告
    document.getElementById('exportReport').addEventListener('click', exportReport);
    
    // 生成依赖图
    document.getElementById('generateGraph').addEventListener('click', generateDependencyGraph);
    
    // 修复漏洞
    document.getElementById('fixVulnerabilities').addEventListener('click', fixAllVulnerabilities);
    
    // 更新依赖
    document.getElementById('updateDeps').addEventListener('click', updateAllDependencies);
}

// 导出报告
async function exportReport() {
    try {
        const response = await fetch(`/api/scan/${scanResult.id}/report`);
        if (!response.ok) throw new Error('导出报告失败');
        
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `scan-report-${scanResult.id}.pdf`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        
        showNotification('报告导出成功', 'success');
    } catch (error) {
        console.error('导出报告失败:', error);
        showNotification('导出报告失败', 'error');
    }
}

// 生成依赖图
function generateDependencyGraph() {
    window.location.href = `/dependencies?id=${scanResult.id}`;
}

// 修复所有漏洞
async function fixAllVulnerabilities() {
    if (!confirm('确定要修复所有漏洞吗？这可能会更新多个依赖版本。')) return;
    
    try {
        const response = await fetch(`/api/scan/${scanResult.id}/fix`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error('修复漏洞失败');
        
        showNotification('开始修复漏洞，请稍候...', 'info');
        
        // 重新加载结果
        setTimeout(() => loadScanResult(scanResult.id), 5000);
        
    } catch (error) {
        console.error('修复漏洞失败:', error);
        showNotification('修复漏洞失败', 'error');
    }
}

// 更新所有依赖
async function updateAllDependencies() {
    if (!confirm('确定要更新所有依赖吗？这可能会影响项目的稳定性。')) return;
    
    try {
        const response = await fetch(`/api/scan/${scanResult.id}/update`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error('更新依赖失败');
        
        showNotification('开始更新依赖，请稍候...', 'info');
        
        // 重新加载结果
        setTimeout(() => loadScanResult(scanResult.id), 5000);
        
    } catch (error) {
        console.error('更新依赖失败:', error);
        showNotification('更新依赖失败', 'error');
    }
}

// 查看依赖详情
function viewDependencyDetails(name) {
    const dep = scanResult.dependencies.find(d => d.name === name);
    if (!dep) return;
    
    // TODO: 显示依赖详情模态框
}

// 检查更新
async function checkUpdates(name) {
    try {
        const response = await fetch(`/api/dependencies/${name}/updates`);
        if (!response.ok) throw new Error('检查更新失败');
        
        const updates = await response.json();
        
        // TODO: 显示更新信息模态框
        
    } catch (error) {
        console.error('检查更新失败:', error);
        showNotification('检查更新失败', 'error');
    }
}

// 查看漏洞详情
function viewVulnerabilityDetails(id) {
    const vuln = scanResult.vulnerabilities.find(v => v.id === id);
    if (!vuln) return;
    
    // TODO: 显示漏洞详情模态框
}

// 修复漏洞
async function fixVulnerability(id) {
    try {
        const response = await fetch(`/api/vulnerabilities/${id}/fix`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error('修复漏洞失败');
        
        showNotification('开始修复漏洞，请稍候...', 'info');
        
        // 重新加载结果
        setTimeout(() => loadScanResult(scanResult.id), 5000);
        
    } catch (error) {
        console.error('修复漏洞失败:', error);
        showNotification('修复漏洞失败', 'error');
    }
}

// 查看冲突详情
function viewConflictDetails(name) {
    const conflict = scanResult.analysis.conflicts.find(c => c.package === name);
    if (!conflict) return;
    
    // TODO: 显示冲突详情模态框
}

// 解决冲突
async function resolveConflict(name) {
    try {
        const response = await fetch(`/api/conflicts/${name}/resolve`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error('解决冲突失败');
        
        showNotification('开始解决冲突，请稍候...', 'info');
        
        // 重新加载结果
        setTimeout(() => loadScanResult(scanResult.id), 5000);
        
    } catch (error) {
        console.error('解决冲突失败:', error);
        showNotification('解决冲突失败', 'error');
    }
}

// 获取状态颜色
function getStatusColor(status) {
    switch (status.toLowerCase()) {
        case 'completed':
            return 'success';
        case 'running':
            return 'primary';
        case 'failed':
            return 'danger';
        case 'stopped':
            return 'warning';
        default:
            return 'secondary';
    }
}

// 获取严重程度颜色
function getSeverityColor(severity) {
    switch (severity.toLowerCase()) {
        case 'high':
            return 'danger';
        case 'medium':
            return 'warning';
        case 'low':
            return 'info';
        default:
            return 'secondary';
    }
}

// 获取推荐版本
function getRecommendedVersion(conflict) {
    // 简单实现：返回最新版本
    return [conflict.current, conflict.required].sort().pop();
}

// 格式化日期
function formatDate(dateString) {
    return new Date(dateString).toLocaleString();
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
// 1. 使用Chart.js绘制图表
// 2. 实现表格排序和过滤
// 3. 支持导出报告功能
// 4. 提供依赖更新建议
// 5. 实现漏洞修复功能 