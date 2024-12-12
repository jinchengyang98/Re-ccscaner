// 全局变量
let ws = null; // WebSocket连接
let currentScanId = null; // 当前扫描ID
let scanStatus = null; // 扫描状态

// 初始化函数
document.addEventListener('DOMContentLoaded', () => {
    // 初始化WebSocket连接
    initWebSocket();
    
    // 初始化表单处理
    initScanForm();
    
    // 初始化文件浏览器
    initFileBrowser();
    
    // 初始化提取器选择
    initExtractorSelection();
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
    if (data.id !== currentScanId) return;
    
    updateScanStatus(data);
}

// 初始化扫描表单
function initScanForm() {
    const form = document.getElementById('scanForm');
    
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        
        // 收集表单数据
        const formData = {
            path: document.getElementById('path').value,
            extractors: Array.from(document.querySelectorAll('input[name="extractors"]:checked')).map(cb => cb.value),
            options: {
                ignoreTests: document.getElementById('ignoreTests').checked,
                includeDevDeps: document.getElementById('includeDevDeps').checked,
                maxDepth: parseInt(document.getElementById('maxDepth').value),
                excludeFiles: document.getElementById('excludeFiles').value.split(',').map(s => s.trim()).filter(Boolean)
            }
        };
        
        try {
            // 显示扫描状态面板
            document.getElementById('scanStatus').style.display = 'block';
            document.getElementById('idleStatus').style.display = 'none';
            
            // 发送扫描请求
            const response = await fetch('/api/scan', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(formData)
            });
            
            if (!response.ok) throw new Error('启动扫描失败');
            
            const result = await response.json();
            currentScanId = result.id;
            
            // 更新UI状态
            updateUIForScanning();
            
            // 添加日志
            addScanLog('开始扫描...');
            
        } catch (error) {
            console.error('启动扫描失败:', error);
            showNotification('启动扫描失败: ' + error.message, 'error');
        }
    });
}

// 初始化文件浏览器
function initFileBrowser() {
    const browseButton = document.getElementById('browsePath');
    
    browseButton.addEventListener('click', async () => {
        try {
            // 调用系统文件选择对话框
            const input = document.createElement('input');
            input.type = 'file';
            input.webkitdirectory = true;
            
            input.addEventListener('change', (event) => {
                const path = event.target.files[0].path;
                document.getElementById('path').value = path;
            });
            
            input.click();
        } catch (error) {
            console.error('选择目录失败:', error);
            showNotification('选择目录失败', 'error');
        }
    });
}

// 初始化提取器选择
function initExtractorSelection() {
    // 添加全选/取消全选功能
    const checkboxes = document.querySelectorAll('input[name="extractors"]');
    
    checkboxes.forEach(checkbox => {
        checkbox.addEventListener('change', () => {
            updateExtractorSelection();
        });
    });
}

// 更新提取器选择状态
function updateExtractorSelection() {
    const selectedCount = document.querySelectorAll('input[name="extractors"]:checked').length;
    const startButton = document.getElementById('startScan');
    
    startButton.disabled = selectedCount === 0;
}

// 更新扫描状态
function updateScanStatus(data) {
    // 更新进度条
    const progressBar = document.getElementById('scanProgress');
    const progressText = document.getElementById('scanProgressText');
    progressBar.style.width = `${data.progress}%`;
    progressText.textContent = `${Math.round(data.progress)}%`;
    
    // 更新状态文本
    const statusText = document.getElementById('scanStatusText');
    statusText.textContent = data.status;
    
    // 更新统计信息
    document.getElementById('foundDeps').textContent = data.dependencies?.length || 0;
    document.getElementById('foundConflicts').textContent = data.analysis?.conflicts?.length || 0;
    document.getElementById('foundVulns').textContent = data.vulnerabilities?.length || 0;
    
    // 添加日志
    if (data.status === 'completed') {
        addScanLog('扫描完成');
        updateUIForCompleted();
    } else if (data.status === 'failed') {
        addScanLog('扫描失败: ' + data.errors.join(', '));
        updateUIForFailed();
    }
}

// 更新UI为扫描中状态
function updateUIForScanning() {
    document.getElementById('startScan').disabled = true;
    document.getElementById('stopScan').disabled = false;
    document.getElementById('scanProgress').classList.add('progress-bar-animated');
}

// 更新UI为完成状态
function updateUIForCompleted() {
    document.getElementById('startScan').disabled = false;
    document.getElementById('stopScan').disabled = true;
    document.getElementById('scanProgress').classList.remove('progress-bar-animated');
}

// 更新UI为失败状态
function updateUIForFailed() {
    document.getElementById('startScan').disabled = false;
    document.getElementById('stopScan').disabled = true;
    document.getElementById('scanProgress').classList.remove('progress-bar-animated');
    document.getElementById('scanStatusText').classList.add('text-danger');
}

// 停止扫描
async function stopScan() {
    if (!currentScanId) return;
    
    try {
        const response = await fetch(`/api/scan/${currentScanId}/stop`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error('停止扫描失败');
        
        addScanLog('扫描已停止');
        updateUIForCompleted();
        
    } catch (error) {
        console.error('停止扫描失败:', error);
        showNotification('停止扫描失败', 'error');
    }
}

// 添加扫描日志
function addScanLog(message) {
    const logs = document.getElementById('scanLogs');
    const timestamp = new Date().toLocaleTimeString();
    logs.innerHTML += `[${timestamp}] ${message}\n`;
    logs.scrollTop = logs.scrollHeight;
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
// 2. 添加文件系统交互
// 3. 实现进度显示
// 4. 提供日志记录
// 5. 支持扫描控制 