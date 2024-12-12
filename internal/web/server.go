package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/your-org/ccscanner/internal/analyzer"
	"github.com/your-org/ccscanner/internal/scanner"
	"github.com/your-org/ccscanner/internal/vulnerability"
	"github.com/your-org/ccscanner/pkg/models"
)

// Server Web服务器
type Server struct {
	router     *mux.Router
	scanner    *scanner.Scanner
	analyzer   *analyzer.DependencyAnalyzer
	detector   *vulnerability.Detector
	upgrader   websocket.Upgrader
	clients    map[*websocket.Conn]bool
	clientsMu  sync.RWMutex
	staticDir  string
	templates  *template.Template
}

// ScanRequest 扫描请求
type ScanRequest struct {
	Path       string   `json:"path"`
	Extractors []string `json:"extractors"`
	Options    struct {
		IgnoreTests    bool     `json:"ignoreTests"`
		ExcludeFiles   []string `json:"excludeFiles"`
		MaxDepth       int      `json:"maxDepth"`
		IncludeDevDeps bool     `json:"includeDevDeps"`
	} `json:"options"`
}

// ScanResult 扫描结果
type ScanResult struct {
	ID           string                 `json:"id"`
	Status       string                 `json:"status"`
	Progress     float64                `json:"progress"`
	Dependencies []*models.Dependency   `json:"dependencies"`
	Analysis     *analyzer.AnalysisResult `json:"analysis"`
	Vulnerabilities []*vulnerability.DetectionResult `json:"vulnerabilities"`
	Errors       []string               `json:"errors"`
	StartTime    time.Time              `json:"startTime"`
	EndTime      time.Time              `json:"endTime"`
	Duration     string                 `json:"duration"`
}

// NewServer 创建新的Web服务器
func NewServer(scanner *scanner.Scanner, analyzer *analyzer.DependencyAnalyzer, detector *vulnerability.Detector, staticDir string) *Server {
	s := &Server{
		router:    mux.NewRouter(),
		scanner:   scanner,
		analyzer:  analyzer,
		detector:  detector,
		staticDir: staticDir,
		clients:   make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
	}

	// 加载模板
	s.templates = template.Must(template.ParseGlob(filepath.Join(staticDir, "templates/*.html")))

	// 注册路由
	s.registerRoutes()

	return s
}

// Start 启动服务器
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

// registerRoutes 注册路由
func (s *Server) registerRoutes() {
	// 静态文件
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(s.staticDir))))

	// HTML页面
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")
	s.router.HandleFunc("/scan", s.handleScanPage).Methods("GET")
	s.router.HandleFunc("/results", s.handleResultsPage).Methods("GET")
	s.router.HandleFunc("/dependencies", s.handleDependenciesPage).Methods("GET")
	s.router.HandleFunc("/vulnerabilities", s.handleVulnerabilitiesPage).Methods("GET")

	// API端点
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/scan", s.handleScan).Methods("POST")
	api.HandleFunc("/scan/{id}", s.handleGetScan).Methods("GET")
	api.HandleFunc("/scan/{id}/stop", s.handleStopScan).Methods("POST")
	api.HandleFunc("/scan/{id}/results", s.handleGetResults).Methods("GET")
	api.HandleFunc("/dependencies", s.handleGetDependencies).Methods("GET")
	api.HandleFunc("/dependencies/graph", s.handleGetDependencyGraph).Methods("GET")
	api.HandleFunc("/dependencies/stats", s.handleGetDependencyStats).Methods("GET")
	api.HandleFunc("/vulnerabilities", s.handleGetVulnerabilities).Methods("GET")
	api.HandleFunc("/vulnerabilities/stats", s.handleGetVulnerabilityStats).Methods("GET")
	api.HandleFunc("/ws", s.handleWebSocket).Methods("GET")

	// 系统信息
	api.HandleFunc("/system/info", s.handleSystemInfo).Methods("GET")
	api.HandleFunc("/system/stats", s.handleSystemStats).Methods("GET")
}

// handleIndex 处理首页请求
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "index.html", nil)
}

// handleScanPage 处理扫描页面请求
func (s *Server) handleScanPage(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "scan.html", nil)
}

// handleResultsPage 处理结果页面请求
func (s *Server) handleResultsPage(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "results.html", nil)
}

// handleDependenciesPage 处理依赖页面请求
func (s *Server) handleDependenciesPage(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "dependencies.html", nil)
}

// handleVulnerabilitiesPage 处理漏洞页面请求
func (s *Server) handleVulnerabilitiesPage(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "vulnerabilities.html", nil)
}

// handleScan 处理扫描请求
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 创建扫描结果
	result := &ScanResult{
		ID:        fmt.Sprintf("scan-%d", time.Now().UnixNano()),
		Status:    "running",
		StartTime: time.Now(),
	}

	// 异步执行扫描
	go func() {
		// 扫描依赖
		deps, err := s.scanner.Scan(req.Path, req.Extractors)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Status = "failed"
			s.broadcastUpdate(result)
			return
		}
		result.Dependencies = deps
		result.Progress = 33.3
		s.broadcastUpdate(result)

		// 分析依赖
		analysis, err := s.analyzer.Analyze(deps)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Status = "failed"
			s.broadcastUpdate(result)
			return
		}
		result.Analysis = analysis
		result.Progress = 66.6
		s.broadcastUpdate(result)

		// 检测漏洞
		vulns, err := s.detector.DetectVulnerabilities(deps)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Status = "failed"
			s.broadcastUpdate(result)
			return
		}
		result.Vulnerabilities = vulns
		result.Progress = 100
		result.Status = "completed"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).String()
		s.broadcastUpdate(result)
	}()

	json.NewEncoder(w).Encode(map[string]string{
		"id": result.ID,
	})
}

// handleGetScan 处理获取扫描状态请求
func (s *Server) handleGetScan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// 获取扫描结果
	result := &ScanResult{
		ID: id,
		// 从存储中获取结果
	}

	json.NewEncoder(w).Encode(result)
}

// handleStopScan 处理停止扫描请求
func (s *Server) handleStopScan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// 停止扫描
	// ...

	json.NewEncoder(w).Encode(map[string]string{
		"status": "stopped",
	})
}

// handleGetResults 处理获取扫描结果请求
func (s *Server) handleGetResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// 获取扫描结果
	result := &ScanResult{
		ID: id,
		// 从存储中获取结果
	}

	json.NewEncoder(w).Encode(result)
}

// handleGetDependencies 处理获取依赖列表请求
func (s *Server) handleGetDependencies(w http.ResponseWriter, r *http.Request) {
	// 获取依赖列表
	// ...

	json.NewEncoder(w).Encode(map[string]interface{}{
		"dependencies": []*models.Dependency{},
	})
}

// handleGetDependencyGraph 处理获取依赖图请求
func (s *Server) handleGetDependencyGraph(w http.ResponseWriter, r *http.Request) {
	// 获取依赖图
	// ...

	json.NewEncoder(w).Encode(map[string]interface{}{
		"graph": map[string]interface{}{},
	})
}

// handleGetDependencyStats 处理获取依赖统计信息请求
func (s *Server) handleGetDependencyStats(w http.ResponseWriter, r *http.Request) {
	// 获取依赖统计信息
	// ...

	json.NewEncoder(w).Encode(map[string]interface{}{
		"stats": map[string]interface{}{},
	})
}

// handleGetVulnerabilities 处理获取漏洞列表请求
func (s *Server) handleGetVulnerabilities(w http.ResponseWriter, r *http.Request) {
	// 获取漏洞列表
	// ...

	json.NewEncoder(w).Encode(map[string]interface{}{
		"vulnerabilities": []*vulnerability.DetectionResult{},
	})
}

// handleGetVulnerabilityStats 处理获取漏洞统计信息请求
func (s *Server) handleGetVulnerabilityStats(w http.ResponseWriter, r *http.Request) {
	// 获取漏洞统计信息
	// ...

	json.NewEncoder(w).Encode(map[string]interface{}{
		"stats": map[string]interface{}{},
	})
}

// handleSystemInfo 处理获取系统信息请求
func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	// 获取系统信息
	// ...

	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":   "1.0.0",
		"buildTime": "2024-01-01",
		"goVersion": "1.21",
	})
}

// handleSystemStats 处理获取系统统计信息请求
func (s *Server) handleSystemStats(w http.ResponseWriter, r *http.Request) {
	// 获取系统统计信息
	// ...

	json.NewEncoder(w).Encode(map[string]interface{}{
		"stats": map[string]interface{}{},
	})
}

// handleWebSocket 处理WebSocket连接
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// 添加客户端
	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	// 移除客户端
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	// 保持连接
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// broadcastUpdate 广播更新消息
func (s *Server) broadcastUpdate(result *ScanResult) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for client := range s.clients {
		err := client.WriteJSON(result)
		if err != nil {
			client.Close()
			delete(s.clients, client)
		}
	}
}

// 注意事项:
// 1. 添加错误处理和日志记录
// 2. 实现结果存储和缓存
// 3. 添加认证和授权
// 4. 添加请求限制和超时
// 5. 实现WebSocket连接管理 