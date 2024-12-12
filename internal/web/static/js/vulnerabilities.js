// 全局变量
let scanResult = null; // 扫描结果
let vulnTypeChart = null; // 漏洞类型图表
let affectedComponentsChart = null; // 受影响组件图表
let currentFilters = {
    search: '',
    severity: '',
    status: '',
    type: ''
}; // 当前过滤条件

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
        updateVulnerabilityStats();
        updateCharts();
        updateVulnerabilityTable();
        
    } catch (error) {
        console.error('加载扫描结果失败:', error);
        showNotification('加载扫描结果失败', 'error');
    }
}

// 初始化图表
function initCharts() {
    // 漏洞类型分布图表
    const vulnTypeCtx = document.getElementById('vulnTypeChart').getContext('2d');
    vulnTypeChart = new Chart(vulnTypeCtx, {
        type: 'pie',
        data: {
            labels: [],
            datasets: [{
                data: [],
                backgroundColor: [
                    '#dc3545',
                    '#ffc107',
                    '#17a2b8',
                    '#6c757d',
                    '#28a745'
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
    
    // 受影响组件分布图表
    const affectedComponentsCtx = document.getElementById('affectedComponentsChart').getContext('2d');
    affectedComponentsChart = new Chart(affectedComponentsCtx, {
        type: 'bar',
        data: {
            labels: [],
            datasets: [{
                label: '漏洞数量',
                data: [],
                backgroundColor: '#007bff'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        stepSize: 1
                    }
                }
            }
        }
    });
}

// 更新漏洞统计
function updateVulnerabilityStats() {
    // 按严重程度统计
    const severityCounts = {
        high: 0,
        medium: 0,
        low: 0
    };
    
    scanResult.vulnerabilities.forEach(vuln => {
        severityCounts[vuln.severity.toLowerCase()]++;
    });
    
    // 更新显示
    document.getElementById('highVulnsCount').textContent = severityCounts.high;
    document.getElementById('mediumVulnsCount').textContent = severityCounts.medium;
    document.getElementById('lowVulnsCount').textContent = severityCounts.low;
    document.getElementById('fixedVulnsCount').textContent = scanResult.vulnerabilities.filter(v => v.status === 'fixed').length;
}

// 更新图表
function updateCharts() {
    // 更新漏洞类型分布
    const vulnTypes = {};
    scanResult.vulnerabilities.forEach(vuln => {
        vulnTypes[vuln.type] = (vulnTypes[vuln.type] || 0) + 1;
    });
    
    vulnTypeChart.data.labels = Object.keys(vulnTypes);
    vulnTypeChart.data.datasets[0].data = Object.values(vulnTypes);
    vulnTypeChart.update();
    
    // 更新受影响组件分布
    const affectedComponents = {};
    scanResult.vulnerabilities.forEach(vuln => {
        affectedComponents[vuln.package] = (affectedComponents[vuln.package] || 0) + 1;
    });
    
    // 按漏洞数量排序
    const sortedComponents = Object.entries(affectedComponents)
        .sort(([,a], [,b]) => b - a)
        .slice(0, 10); // 只显示前10个
    
    affectedComponentsChart.data.labels = sortedComponents.map(([name]) => name);
    affectedComponentsChart.data.datasets[0].data = sortedComponents.map(([,count]) => count);
    affectedComponentsChart.update();
}

// 更新漏洞表格
function updateVulnerabilityTable() {
    const tbody = document.getElementById('vulnsTable').getElementsByTagName('tbody')[0];
    tbody.innerHTML = '';
    
    // 应用过滤器
    const filteredVulns = filterVulnerabilities(scanResult.vulnerabilities);
    
    // 更新表格内容
    filteredVulns.forEach(vuln => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>
                <input type="checkbox" class="form-check-input" value="${vuln.id}">
            </td>
            <td>${vuln.id}</td>
            <td>${vuln.package}</td>
            <td>${vuln.version}</td>
            <td>
                <span class="badge bg-${getSeverityColor(vuln.severity)}">
                    ${vuln.severity}
                </span>
            </td>
            <td>${truncateText(vuln.description, 100)}</td>
            <td>${vuln.status}</td>
            <td>${vuln.fixedVersion || '无'}</td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" onclick="viewVulnerabilityDetails('${vuln.id}')">
                        <i class="fas fa-info-circle"></i>
                    </button>
                    <button class="btn btn-outline-success" onclick="fixVulnerability('${vuln.id}')">
                        <i class="fas fa-wrench"></i>
                    </button>
                    <button class="btn btn-outline-warning" onclick="ignoreVulnerability('${vuln.id}')">
                        <i class="fas fa-ban"></i>
                    </button>
                </div>
            </td>
        `;
        tbody.appendChild(tr);
    });
    
    // 更新分页
    updatePagination(filteredVulns.length);
}

// 过滤漏洞
function filterVulnerabilities(vulnerabilities) {
    return vulnerabilities.filter(vuln => {
        // 搜索过滤
        if (currentFilters.search && !matchesSearch(vuln, currentFilters.search)) {
            return false;
        }
        
        // 严重程度过滤
        if (currentFilters.severity && vuln.severity.toLowerCase() !== currentFilters.severity) {
            return false;
        }
        
        // 状态过滤
        if (currentFilters.status && vuln.status !== currentFilters.status) {
            return false;
        }
        
        // 类型过滤
        if (currentFilters.type && vuln.type !== currentFilters.type) {
            return false;
        }
        
        return true;
    });
}

// 搜索匹配
function matchesSearch(vuln, query) {
    query = query.toLowerCase();
    return (
        vuln.id.toLowerCase().includes(query) ||
        vuln.package.toLowerCase().includes(query) ||
        vuln.description.toLowerCase().includes(query)
    );
}

// 更新分页
function updatePagination(totalItems) {
    const itemsPerPage = 10;
    const totalPages = Math.ceil(totalItems / itemsPerPage);
    
    const pagination = document.getElementById('vulnsPagination');
    pagination.innerHTML = '';
    
    // 添加分页按钮
    for (let i = 1; i <= totalPages; i++) {
        const li = document.createElement('li');
        li.className = 'page-item';
        li.innerHTML = `
            <button class="page-link" onclick="goToPage(${i})">${i}</button>
        `;
        pagination.appendChild(li);
    }
}

// 初始化事件处理
function initEventHandlers() {
    // 搜索框
    document.getElementById('searchVulns').addEventListener('input', (e) => {
        currentFilters.search = e.target.value;
        updateVulnerabilityTable();
    });
    
    // 过滤器
    document.getElementById('severityFilter').addEventListener('change', (e) => {
        currentFilters.severity = e.target.value;
        updateVulnerabilityTable();
    });
    
    document.getElementById('statusFilter').addEventListener('change', (e) => {
        currentFilters.status = e.target.value;
        updateVulnerabilityTable();
    });
    
    document.getElementById('typeFilter').addEventListener('change', (e) => {
        currentFilters.type = e.target.value;
        updateVulnerabilityTable();
    });
    
    // 清除过滤器
    document.getElementById('clearFilters').addEventListener('click', () => {
        currentFilters = {
            search: '',
            severity: '',
            status: '',
            type: ''
        };
        
        // 重置表单
        document.getElementById('searchVulns').value = '';
        document.getElementById('severityFilter').value = '';
        document.getElementById('statusFilter').value = '';
        document.getElementById('typeFilter').value = '';
        
        updateVulnerabilityTable();
    });
    
    // 全选框
    document.getElementById('selectAll').addEventListener('change', (e) => {
        const checkboxes = document.querySelectorAll('#vulnsTable tbody input[type="checkbox"]');
        checkboxes.forEach(cb => cb.checked = e.target.checked);
    });
    
    // 批量修复按钮
    document.getElementById('batchFix').addEventListener('click', () => {
        const selectedIds = getSelectedVulnerabilityIds();
        if (selectedIds.length === 0) {
            showNotification('请选择要修复的漏洞', 'warning');
            return;
        }
        
        if (confirm(`确定要修复选中的 ${selectedIds.length} 个漏洞吗？`)) {
            fixVulnerabilities(selectedIds);
        }
    });
    
    // 导出报告按钮
    document.getElementById('exportReport').addEventListener('click', exportVulnerabilityReport);
}

// 获取选中的漏洞ID
function getSelectedVulnerabilityIds() {
    const checkboxes = document.querySelectorAll('#vulnsTable tbody input[type="checkbox"]:checked');
    return Array.from(checkboxes).map(cb => cb.value);
}

// 查看漏洞详情
function viewVulnerabilityDetails(id) {
    const vuln = scanResult.vulnerabilities.find(v => v.id === id);
    if (!vuln) return;
    
    // 更新模态框内容
    document.getElementById('vulnId').textContent = vuln.id;
    document.getElementById('vulnCVE').textContent = vuln.cve || '无';
    document.getElementById('vulnCWE').textContent = vuln.cwe || '无';
    document.getElementById('vulnDiscovered').textContent = formatDate(vuln.discoveredDate);
    document.getElementById('vulnStatus').textContent = vuln.status;
    
    document.getElementById('vulnComponent').textContent = vuln.package;
    document.getElementById('vulnVersion').textContent = vuln.version;
    document.getElementById('vulnFixedVersion').textContent = vuln.fixedVersion || '无';
    document.getElementById('vulnDepType').textContent = vuln.dependencyType;
    document.getElementById('vulnDepPath').textContent = vuln.path.join(' → ');
    
    document.getElementById('vulnDescription').textContent = vuln.description;
    document.getElementById('vulnRemediation').textContent = vuln.remediation;
    
    // 更新参考链接
    const referencesList = document.getElementById('vulnReferences');
    referencesList.innerHTML = '';
    vuln.references.forEach(ref => {
        const li = document.createElement('li');
        li.innerHTML = `<a href="${ref}" target="_blank">${ref}</a>`;
        referencesList.appendChild(li);
    });
    
    // 显示模态框
    const modal = new bootstrap.Modal(document.getElementById('vulnDetailsModal'));
    modal.show();
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

// 批量修复漏洞
async function fixVulnerabilities(ids) {
    try {
        const response = await fetch('/api/vulnerabilities/fix', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ ids })
        });
        
        if (!response.ok) throw new Error('批量修复漏洞失败');
        
        showNotification('开始修复漏洞，请稍候...', 'info');
        
        // 重新加载结果
        setTimeout(() => loadScanResult(scanResult.id), 5000);
        
    } catch (error) {
        console.error('批量修复漏洞失败:', error);
        showNotification('批量修复漏洞失败', 'error');
    }
}

// 忽略漏洞
async function ignoreVulnerability(id) {
    if (!confirm('确定要忽略这个漏洞吗？')) return;
    
    try {
        const response = await fetch(`/api/vulnerabilities/${id}/ignore`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error('忽略漏洞失败');
        
        showNotification('已忽略漏洞', 'success');
        
        // 重新加载结果
        loadScanResult(scanResult.id);
        
    } catch (error) {
        console.error('忽略漏洞失败:', error);
        showNotification('忽略漏洞失败', 'error');
    }
}

// 导出漏洞报告
async function exportVulnerabilityReport() {
    try {
        const response = await fetch(`/api/scan/${scanResult.id}/vulnerability-report`);
        if (!response.ok) throw new Error('导出报告失败');
        
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `vulnerability-report-${scanResult.id}.pdf`;
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

// 截断文本
function truncateText(text, maxLength) {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
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
// 2. 实现漏洞过滤和搜索
// 3. 支持批量操作功能
// 4. 提供详细的漏洞信息
// 5. 实现漏洞修复流程 