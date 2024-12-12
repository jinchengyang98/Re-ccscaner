// 全局变量
let ws = null; // WebSocket连接
let stats = {}; // 统计数据
let recentScans = []; // 最近扫描记录

// 初始化函数
document.addEventListener('DOMContentLoaded', () => {
    // 初始化WebSocket连接
    initWebSocket();
    
    // 加载统计数据
    loadStats();
    
    // 加载最近扫描记录
    loadRecentScans();
    
    // 设置自动刷新
    setInterval(() => {
        loadStats();
        loadRecentScans();
    }, 30000); // 每30秒刷新一次
});

// 初始化WebSocket连接
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/ws`;
    
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
        console.log('WebSocket连接已建立');
    };
    
    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        handleWebSocketMessage(data);
    };
    
    ws.onclose = () => {
        console.log('WebSocket连接已关闭');
        // 5秒后尝试重新连接
        setTimeout(initWebSocket, 5000);
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket错误:', error);
    };
}

// 处理WebSocket消息
function handleWebSocketMessage(data) {
    switch (data.type) {
        case 'stats_update':
            updateStats(data.stats);
            break;
        case 'scan_complete':
            updateRecentScans();
            break;
        case 'notification':
            showNotification(data.message, data.level);
            break;
    }
}

// 加载统计数据
async function loadStats() {
    try {
        const response = await fetch('/api/system/stats');
        if (!response.ok) throw new Error('获取统计数据失败');
        
        stats = await response.json();
        updateStatsDisplay();
    } catch (error) {
        console.error('加载统计数据失败:', error);
        showNotification('加载统计数据失败', 'error');
    }
}

// 更新统计数据显示
function updateStats(newStats) {
    stats = newStats;
    updateStatsDisplay();
}

// 更新统计显示
function updateStatsDisplay() {
    // 更新总扫描次数
    document.getElementById('totalScans').textContent = stats.totalScans || 0;
    
    // 更新发现依赖数
    document.getElementById('totalDeps').textContent = stats.totalDeps || 0;
    
    // 更新发现漏洞数
    document.getElementById('totalVulns').textContent = stats.totalVulns || 0;
    
    // 更新版本冲突数
    document.getElementById('totalConflicts').textContent = stats.totalConflicts || 0;
}

// 加载最近扫描记录
async function loadRecentScans() {
    try {
        const response = await fetch('/api/scan?limit=5'); // 获取最近5条记录
        if (!response.ok) throw new Error('获取扫描记录失败');
        
        recentScans = await response.json();
        updateRecentScansDisplay();
    } catch (error) {
        console.error('加载扫描记录失败:', error);
        showNotification('加载扫描记录失败', 'error');
    }
}

// 更新扫描记录
function updateRecentScans() {
    loadRecentScans();
}

// 更新扫描记录显示
function updateRecentScansDisplay() {
    const tbody = document.getElementById('recentScans');
    tbody.innerHTML = '';
    
    recentScans.forEach(scan => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${scan.id}</td>
            <td>${scan.path}</td>
            <td>
                <span class="badge ${getStatusBadgeClass(scan.status)}">
                    ${scan.status}
                </span>
            </td>
            <td>${scan.dependencies?.length || 0}</td>
            <td>${scan.vulnerabilities?.length || 0}</td>
            <td>${formatDate(scan.startTime)}</td>
            <td>
                <div class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" onclick="viewScanDetails('${scan.id}')">
                        <i class="fas fa-eye"></i>
                    </button>
                    <button class="btn btn-outline-success" onclick="exportScanReport('${scan.id}')">
                        <i class="fas fa-download"></i>
                    </button>
                    <button class="btn btn-outline-danger" onclick="deleteScan('${scan.id}')">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </td>
        `;
        tbody.appendChild(tr);
    });
}

// 获取状态徽章样式
function getStatusBadgeClass(status) {
    switch (status.toLowerCase()) {
        case 'completed':
            return 'bg-success';
        case 'running':
            return 'bg-primary';
        case 'failed':
            return 'bg-danger';
        case 'stopped':
            return 'bg-warning';
        default:
            return 'bg-secondary';
    }
}

// 格式化日期
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString();
}

// 查看扫描详情
function viewScanDetails(scanId) {
    window.location.href = `/results?id=${scanId}`;
}

// 导出扫描报告
async function exportScanReport(scanId) {
    try {
        const response = await fetch(`/api/scan/${scanId}/report`);
        if (!response.ok) throw new Error('导出报告失败');
        
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `scan-report-${scanId}.pdf`;
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

// 删除扫描记录
async function deleteScan(scanId) {
    if (!confirm('确定要删除这条扫描记录吗？')) return;
    
    try {
        const response = await fetch(`/api/scan/${scanId}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) throw new Error('删除扫描记录失败');
        
        showNotification('删除扫描记录成功', 'success');
        loadRecentScans();
    } catch (error) {
        console.error('删除扫描记录失败:', error);
        showNotification('删除扫描记录失败', 'error');
    }
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
// 1. 实现WebSocket实时更新
// 2. 添加错误处理和重试机制
// 3. 实现数据自动刷新
// 4. 提供用户友好的通知
// 5. 支持导出报告功能 