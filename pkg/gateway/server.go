package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeroclaw-labs/goclaw/pkg/agent"
	"github.com/zeroclaw-labs/goclaw/pkg/auth"
	"github.com/zeroclaw-labs/goclaw/pkg/cron"
	"github.com/zeroclaw-labs/goclaw/pkg/integrations"
	"github.com/zeroclaw-labs/goclaw/pkg/memory"
	"github.com/zeroclaw-labs/goclaw/pkg/session"
	"github.com/zeroclaw-labs/goclaw/pkg/tools"
	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

type Server struct {
	addr           string
	server         *http.Server
	agent          *agent.Agent
	staticDir      string
	staticFS       http.FileSystem
	sseClients     map[string]chan *types.StreamChunk
	sseClientsMu   sync.RWMutex
	authMiddleware func(http.Handler) http.Handler
	wsClients      map[string]*wsClient
	wsClientsMu    sync.RWMutex
	config         map[string]interface{}
	memoryBackend  interface{}
	authService    *auth.AuthService
	userManager    *auth.UserManager
	pairingGuard   PairingGuard
	scheduler      *cron.Scheduler
	sessionManager *session.Manager
}

// PairingGuard 配对码守卫接口
type PairingGuard interface {
	PairingCode() string
	VerifyCode(code string) bool
	IsEnabled() bool
}

type wsClient struct {
	conn     *websocket.Conn
	sendChan chan []byte
}

type Config struct {
	Addr           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	StaticDir      string // Static files directory
	StaticFS       http.FileSystem
}

type prefixFS struct {
	fs     http.FileSystem
	prefix string
}

func (w *prefixFS) Open(name string) (http.File, error) {
	return w.fs.Open(w.prefix + name)
}

func NewServer(addr string, agent *agent.Agent, staticDir string) *Server {
	return &Server{
		addr:       addr,
		agent:      agent,
		staticDir:  staticDir,
		sseClients: make(map[string]chan *types.StreamChunk),
		wsClients:  make(map[string]*wsClient),
		config:     make(map[string]interface{}),
	}
}

func NewServerWithFS(addr string, agent *agent.Agent, staticFS http.FileSystem) *Server {
	return &Server{
		addr:       addr,
		agent:      agent,
		staticFS:   staticFS,
		sseClients: make(map[string]chan *types.StreamChunk),
		wsClients:  make(map[string]*wsClient),
		config:     make(map[string]interface{}),
	}
}

func (s *Server) SetAuthService(authService *auth.AuthService) {
	s.authService = authService
}

func (s *Server) SetUserManager(userManager *auth.UserManager) {
	s.userManager = userManager
}

func (s *Server) SetScheduler(scheduler *cron.Scheduler) {
	s.scheduler = scheduler
}

func (s *Server) SetPairingGuard(guard PairingGuard) {
	s.pairingGuard = guard
}

func (s *Server) SetConfig(key string, value interface{}) {
	if s.config == nil {
		s.config = make(map[string]interface{})
	}
	s.config[key] = value
}

func (s *Server) SetMemoryBackend(backend interface{}) {
	s.memoryBackend = backend
}

func (s *Server) SetSessionManager(manager *session.Manager) {
	s.sessionManager = manager
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// Handle API routes FIRST (before static files)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/api/v1/completions", s.handleCompletions)
	mux.HandleFunc("/api/v1/models", s.handleModels)
	mux.HandleFunc("/api/v1/embeddings", s.handleEmbeddings)
	mux.HandleFunc("/api/sse", s.handleSSE)
	
	// WeChat login routes
	mux.HandleFunc("/api/auth/wechat/login", s.handleWechatLogin)
	mux.HandleFunc("/api/auth/wechat/callback", s.handleWechatCallback)
	mux.HandleFunc("/api/auth/wechat/user", s.handleWechatUserInfo)
	
	// Admin routes
	mux.HandleFunc("/api/auth/admin/login", s.handleAdminLogin)
	mux.HandleFunc("/api/auth/admin/users", s.handleAdminUsers)
	mux.HandleFunc("/api/auth/admin/users/approve", s.handleAdminApproveUser)
	mux.HandleFunc("/api/auth/admin/password", s.handleAdminPasswordChange)
	
	// User routes
	mux.HandleFunc("/api/auth/user/info", s.handleUserInfo)
	mux.HandleFunc("/api/auth/user/update", s.handleUserUpdate)
	
	// Session management routes
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSessionDetail)
	
	// WebSocket route for agent chat
	mux.HandleFunc("/api/ws/chat", s.handleWebSocket)
	mux.HandleFunc("/api/ws", s.handleWebSocket)
	
	// Pairing route
	mux.HandleFunc("/api/pair", s.handlePair)
	
	// Generic /api/* handler (handles all other /api/* paths)
	mux.HandleFunc("/api/", s.handleAPI)
	
	// Serve static files from configured directory or embedded filesystem
	var staticFS http.FileSystem
	
	// Prefer embedded filesystem if provided
	if s.staticFS != nil {
		staticFS = s.staticFS
		log.Printf("Using embedded static files")
	} else if s.staticDir != "" {
		if _, err := os.Stat(s.staticDir); err == nil {
			staticFS = http.Dir(s.staticDir)
			log.Printf("Using static files from: %s", s.staticDir)
		}
	}
	
	if staticFS != nil {
		// Handle static assets under /assets/ - map to web/dist/assets/
		mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(&prefixFS{
			fs:     staticFS,
			prefix: "web/dist/assets",
		})))
		
		// SPA fallback: serve index.html for root and any non-API, non-static paths
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// If it's a file request (has extension), try to serve it
			if filepath.Ext(r.URL.Path) != "" {
				// Try to open the file
				f, err := staticFS.Open("web/dist" + r.URL.Path)
				if err == nil {
					defer f.Close()
					// Determine content type
					contentType := "application/octet-stream"
					switch filepath.Ext(r.URL.Path) {
					case ".html":
						contentType = "text/html; charset=utf-8"
					case ".css":
						contentType = "text/css; charset=utf-8"
					case ".js":
						contentType = "application/javascript; charset=utf-8"
					case ".json":
						contentType = "application/json; charset=utf-8"
					case ".png":
						contentType = "image/png"
					case ".jpg", ".jpeg":
						contentType = "image/jpeg"
					case ".svg":
						contentType = "image/svg+xml"
					}
					w.Header().Set("Content-Type", contentType)
					http.ServeContent(w, r, r.URL.Path, time.Time{}, f.(http.File))
					return
				}
			}
			// Otherwise serve index.html for SPA
			f, err := staticFS.Open("web/dist/index.html")
			if err != nil {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			defer f.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, "/index.html", time.Time{}, f.(http.File))
		})
		
		log.Printf("Available at: http://localhost%s/", s.addr)
	}

	var handler http.Handler = mux
	if s.authMiddleware != nil {
		handler = s.authMiddleware(mux)
	}

	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Gateway starting on %s", s.addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Gateway error: %v", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Return health status matching ZeroClaw format
	response := map[string]interface{}{
		"status":           "ok",
		"paired":           false,
		"require_pairing":  false,
		"runtime": map[string]interface{}{
			"pid":           os.Getpid(),
			"updated_at":    time.Now().Format(time.RFC3339),
			"uptime_seconds": 0,
			"components": map[string]interface{}{
				"gateway": map[string]interface{}{
					"status":        "ok",
					"updated_at":    time.Now().Format(time.RFC3339),
					"last_ok":       time.Now().Format(time.RFC3339),
					"last_error":    nil,
					"restart_count": 0,
				},
				"daemon": map[string]interface{}{
					"status":        "ok",
					"updated_at":    time.Now().Format(time.RFC3339),
					"last_ok":       time.Now().Format(time.RFC3339),
					"last_error":    nil,
					"restart_count": 0,
				},
				"channels": map[string]interface{}{
					"status":        "ok",
					"updated_at":    time.Now().Format(time.RFC3339),
					"last_ok":       time.Now().Format(time.RFC3339),
					"last_error":    nil,
					"restart_count": 0,
				},
				"scheduler": map[string]interface{}{
					"status":        "ok",
					"updated_at":    time.Now().Format(time.RFC3339),
					"last_ok":       time.Now().Format(time.RFC3339),
					"last_error":    nil,
					"restart_count": 0,
				},
			},
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	
	// 检查是否启用了配对
	if s.pairingGuard == nil || !s.pairingGuard.IsEnabled() {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "pairing not enabled"})
		return
	}
	
	// 支持两种方式获取配对码：header 或 JSON body
	code := r.Header.Get("X-Pairing-Code")
	if code == "" {
		var req struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}
		code = req.Code
	}
	
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "pairing code required"})
		return
	}
	
	if s.pairingGuard.VerifyCode(code) {
		// 生成一个简单的 token
		token := fmt.Sprintf("paired_%d", time.Now().Unix())
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "pairing successful",
			"token":   token,
		})
		return
	}
	
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "invalid pairing code"})
}

type ChatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []types.ChatMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	Tools       []*types.ToolSpec   `json:"tools,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []ChatChoice      `json:"choices"`
	Usage   *types.TokenUsage `json:"usage,omitempty"`
}

type ChatChoice struct {
	Index        int               `json:"index"`
	Message      types.ChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = "gpt-4o"
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	if req.Stream {
		s.handleStreamingChat(w, r, &req, model, temperature)
		return
	}

	var lastMessage string
	for _, msg := range req.Messages {
		if msg.Role == types.RoleUser {
			lastMessage = msg.Content
		}
	}

	response, err := s.agent.ProcessMessage(r.Context(), lastMessage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	finishReason := "stop"
	if response.HasToolCalls() {
		finishReason = "tool_calls"
	}

	chatResp := ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []ChatChoice{
			{
				Index: 0,
				Message: types.ChatMessage{
					Role:    types.RoleAssistant,
					Content: response.TextOrEmpty(),
				},
				FinishReason: finishReason,
			},
		},
		Usage: response.Usage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResp)
}

func (s *Server) handleStreamingChat(w http.ResponseWriter, r *http.Request, req *ChatCompletionRequest, model string, temperature float64) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	clientID := fmt.Sprintf("sse_%d", time.Now().UnixNano())
	ch := make(chan *types.StreamChunk, 10)

	s.sseClientsMu.Lock()
	s.sseClients[clientID] = ch
	s.sseClientsMu.Unlock()

	defer func() {
		s.sseClientsMu.Lock()
		delete(s.sseClients, clientID)
		s.sseClientsMu.Unlock()
		close(ch)
	}()

	go func() {
		var lastMessage string
		for _, msg := range req.Messages {
			if msg.Role == types.RoleUser {
				lastMessage = msg.Content
			}
		}
		_, _ = s.agent.ProcessMessage(r.Context(), lastMessage)
	}()

	for chunk := range ch {
		if chunk.IsFinal {
			break
		}

		data := map[string]interface{}{
			"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]string{
						"content": chunk.Delta,
					},
				},
			},
		}

		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (s *Server) handleCompletions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"message": "completions not implemented, use chat/completions",
			"type":    "invalid_request_error",
		},
	})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"id":       "gpt-4o",
				"object":   "model",
				"created":  1677610602,
				"owned_by": "openai",
			},
		},
	})
}

func (s *Server) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   []interface{}{},
	})
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	s.handleChatCompletions(w, r)
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	// Extract the path after /api/
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	
	w.Header().Set("Content-Type", "application/json")
	
	// Handle memory-related paths first
	if strings.HasPrefix(path, "memory") {
		// Remove "memory" prefix and split the rest
		restPath := strings.TrimPrefix(path, "memory")
		memoryKey := ""
		if restPath != "" {
			restPath = strings.TrimPrefix(restPath, "/")
			if restPath != "" {
				memoryKey = restPath
			}
		}
		
		// Handle recall endpoint
		if memoryKey == "recall" {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
				return
			}

			if s.memoryBackend == nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "memory backend not available"})
				return
			}

			var req struct {
				Query    string  `json:"query"`
				Limit    int     `json:"limit"`
				Category *string `json:"category,omitempty"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			if req.Query == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "query is required"})
				return
			}

			if req.Limit <= 0 {
				req.Limit = 5
			}

			if mb, ok := s.memoryBackend.(interface {
				Recall(ctx context.Context, query string, limit int, category *string) ([]agent.MemoryEntry, error)
			}); ok {
				entries, err := mb.Recall(r.Context(), req.Query, req.Limit, req.Category)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
					return
				}

				response := map[string]interface{}{
					"success": true,
					"output":  formatMemoryEntries(entries),
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "memory backend not available"})
			return
		}
		
		if r.Method == http.MethodGet && memoryKey == "" {
			// Return all memory entries
			entries := []map[string]interface{}{}
			
			if s.memoryBackend != nil {
				if mb, ok := s.memoryBackend.(interface {
					List(ctx context.Context, category *string) ([]map[string]interface{}, error)
				}); ok {
					memEntries, err := mb.List(r.Context(), nil)
					if err == nil {
						entries = memEntries
					}
				}
			}
			
			response := map[string]interface{}{
				"entries": entries,
				"count":   len(entries),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		
		if r.Method == http.MethodPost && memoryKey == "" {
			// Store a new memory entry
			if r.Body == nil || r.ContentLength == 0 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "request body is required"})
				return
			}

			if s.memoryBackend == nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "memory backend not available"})
				return
			}

			var req struct {
				Key      string                 `json:"key"`
				Content  string                 `json:"content"`
				Category string                 `json:"category"`
				Metadata map[string]interface{} `json:"metadata,omitempty"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			// Validate required fields
			if req.Key == "" || req.Content == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "key and content are required"})
				return
			}

			// Try to store using agent.Memory interface
			if mb, ok := s.memoryBackend.(interface {
				Store(ctx context.Context, key, content string, category *string, metadata map[string]string) error
			}); ok {
				// Convert metadata
				metadata := make(map[string]string)
				for k, v := range req.Metadata {
					metadata[k] = fmt.Sprintf("%v", v)
				}

				category := &req.Category
				if req.Category == "" {
					category = nil
				}

				if err := mb.Store(r.Context(), req.Key, req.Content, category, metadata); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
					return
				}

				response := map[string]interface{}{
					"status": "success",
					"key":    req.Key,
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "memory backend not available"})
			return
		}
		
		if r.Method == http.MethodDelete && memoryKey != "" {
			if s.memoryBackend != nil {
				if mb, ok := s.memoryBackend.(interface {
					Forget(ctx context.Context, key string) error
				}); ok {
					err := mb.Forget(r.Context(), memoryKey)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
						return
					}
					
					response := map[string]interface{}{
						"status":  "success",
						"deleted": true,
					}
					json.NewEncoder(w).Encode(response)
					return
				}
			}
			
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "memory backend not available"})
			return
		}
		
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid memory request"})
		return
	}
	
	// Handle cron-related paths
	if strings.HasPrefix(path, "cron") {
		// Remove "cron" prefix and split the rest
		restPath := strings.TrimPrefix(path, "cron")
		jobID := ""
		if restPath != "" {
			restPath = strings.TrimPrefix(restPath, "/")
			if restPath != "" {
				jobID = restPath
			}
		}
		
		if r.Method == http.MethodGet && jobID == "" {
			// Return all cron jobs
			if s.scheduler == nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "scheduler not available"})
				return
			}
			
			jobs := s.scheduler.ListJobs()
			response := map[string]interface{}{
				"jobs": jobs,
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		
		if r.Method == http.MethodPost && jobID == "" {
			// Add a new cron job
			if s.scheduler == nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "scheduler not available"})
				return
			}
			
			var req struct {
				Name       string `json:"name,omitempty"`
				Expression string `json:"expression"`
				Command    string `json:"command"`
				Enabled    bool   `json:"enabled"`
			}
			
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			
			if req.Expression == "" || req.Command == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "expression and command are required"})
				return
			}
			
			job := &cron.Job{
				Name:       req.Name,
				Expression: req.Expression,
				Command:    req.Command,
				Enabled:    req.Enabled,
			}
			
			if err := s.scheduler.AddJob(job); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			
			response := map[string]interface{}{
				"status": "ok",
				"job":    job,
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		
		if r.Method == http.MethodDelete && jobID != "" {
			// Delete a cron job
			if s.scheduler == nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "scheduler not available"})
				return
			}
			
			if err := s.scheduler.RemoveJob(jobID); err != nil {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "job not found"})
				return
			}
			
			response := map[string]interface{}{
				"status":  "success",
				"deleted": true,
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid cron request"})
		return
	}
	
	switch path {
	case "status":
		// Return system status matching ZeroClaw format
		provider := "custom:https://ai.gitee.com/v1"
		if p, ok := s.config["provider"].(string); ok {
			provider = p
		}
		
		model := "GLM-4.7-Flash"
		if m, ok := s.config["model"].(string); ok {
			model = m
		}
		
		memoryBackend := "none"
		if mb, ok := s.config["memory_backend"].(string); ok {
			memoryBackend = mb
		}
		
		temperature := 0.7
		if t, ok := s.config["temperature"].(float64); ok {
			temperature = t
		}
		
				wechatEnabled := false
	if we, ok := s.config["wechat_enabled"].(bool); ok {
			wechatEnabled = we
	}
		
	paired := false
	pairingCode := ""
		// 检查是否启用了 require_pairing
	if rp, ok := s.config["require_pairing"].(bool); ok && rp {
			paired = true
	}
		// 检查是否配置了 paired_tokens
	if pt, ok := s.config["paired_tokens"].([]string); ok && len(pt) > 0 {
		paired = true
	}
	// 获取配对码
	if s.pairingGuard != nil && s.pairingGuard.IsEnabled() {
		pairingCode = s.pairingGuard.PairingCode()
	}
	
	response := map[string]interface{}{
		"provider":       provider,
		"model":          model,
		"temperature":    temperature,
		"uptime_seconds": 0,
		"gateway_port":   4096,
		"locale":         "zh-CN",
		"memory_backend": memoryBackend,
		"paired":         paired,
		"pairing_code":   pairingCode,
		"channels":       map[string]bool{},
		"wechatlogin":  wechatEnabled,
			"health": map[string]interface{}{
				"pid":           os.Getpid(),
				"updated_at":    time.Now().Format(time.RFC3339),
				"uptime_seconds": 0,
				"components": map[string]interface{}{
					"gateway": map[string]interface{}{
						"status":        "ok",
						"updated_at":    time.Now().Format(time.RFC3339),
						"last_ok":       time.Now().Format(time.RFC3339),
						"last_error":    nil,
						"restart_count": 0,
					},
					"daemon": map[string]interface{}{
						"status":        "ok",
						"updated_at":    time.Now().Format(time.RFC3339),
						"last_ok":       time.Now().Format(time.RFC3339),
						"last_error":    nil,
						"restart_count": 0,
					},
					"channels": map[string]interface{}{
						"status":        "ok",
						"updated_at":    time.Now().Format(time.RFC3339),
						"last_ok":       time.Now().Format(time.RFC3339),
						"last_error":    nil,
						"restart_count": 0,
					},
					"scheduler": map[string]interface{}{
						"status":        "ok",
						"updated_at":    time.Now().Format(time.RFC3339),
						"last_ok":       time.Now().Format(time.RFC3339),
						"last_error":    nil,
						"restart_count": 0,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
		return
		
	case "cost":
		// Return cost summary
		response := map[string]interface{}{
			"cost": map[string]interface{}{
				"by_model":         map[string]interface{}{},
				"daily_cost_usd":   0.0,
				"monthly_cost_usd": 0.0,
				"request_count":    0,
				"session_cost_usd": 0.0,
				"total_tokens":     0,
			},
		}
		json.NewEncoder(w).Encode(response)
		return
		
	case "config":
		if r.Method == http.MethodGet {
			// Read actual config file
			configPath := os.ExpandEnv("$HOME/.goclaw/config.toml")
			content, err := os.ReadFile(configPath)
			if err != nil {
				// Return default if config doesn't exist
				response := map[string]interface{}{
					"format":  "toml",
					"content": "# GoClaw Configuration\n# Config file not found\n",
				}
				json.NewEncoder(w).Encode(response)
				return
			}
			
			// Mask sensitive fields
			maskedContent := maskSensitiveFields(string(content))
			
			response := map[string]interface{}{
				"format":  "toml",
				"content": maskedContent,
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
		
	case "tools":
		// Return registered tools from agent
		toolSpecs := s.agent.ToolSpecs()
		toolsList := make([]map[string]interface{}, len(toolSpecs))
		for i, spec := range toolSpecs {
			toolsList[i] = map[string]interface{}{
				"name":        spec.Name,
				"description": spec.Description,
				"parameters":  spec.Parameters,
			}
		}
		response := map[string]interface{}{
			"tools": toolsList,
		}
		json.NewEncoder(w).Encode(response)
		return
		
	case "integrations":
		// Return all integrations
		response := map[string]interface{}{
			"integrations": integrations.GetAllIntegrations(),
		}
		json.NewEncoder(w).Encode(response)
		return
		
	case "doctor":
		// Return diagnostics
		response := map[string]interface{}{
			"results": []map[string]interface{}{},
			"summary": map[string]int{
				"ok":       0,
				"warnings": 0,
				"errors":   0,
			},
		}
		json.NewEncoder(w).Encode(response)
		return
		
	case "cli-tools":
		// Return discovered CLI tools
		cliTools := tools.DiscoverCliTools(nil, nil)
		response := map[string]interface{}{
			"cli_tools": cliTools,
		}
		json.NewEncoder(w).Encode(response)
		return
		
	case "health":
		// Return health snapshot matching ZeroClaw format
		response := map[string]interface{}{
			"health": map[string]interface{}{
				"pid":           os.Getpid(),
				"updated_at":    time.Now().Format(time.RFC3339),
				"uptime_seconds": 0,
				"components": map[string]interface{}{
					"gateway": map[string]interface{}{
						"status":        "ok",
						"updated_at":    time.Now().Format(time.RFC3339),
						"last_ok":       time.Now().Format(time.RFC3339),
						"last_error":    nil,
						"restart_count": 0,
					},
					"daemon": map[string]interface{}{
						"status":        "ok",
						"updated_at":    time.Now().Format(time.RFC3339),
						"last_ok":       time.Now().Format(time.RFC3339),
						"last_error":    nil,
						"restart_count": 0,
					},
					"channels": map[string]interface{}{
						"status":        "ok",
						"updated_at":    time.Now().Format(time.RFC3339),
						"last_ok":       time.Now().Format(time.RFC3339),
						"last_error":    nil,
						"restart_count": 0,
					},
					"scheduler": map[string]interface{}{
						"status":        "ok",
						"updated_at":    time.Now().Format(time.RFC3339),
						"last_ok":       time.Now().Format(time.RFC3339),
						"last_error":    nil,
						"restart_count": 0,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Handle /api/chat path
	if path == "chat" && r.Method == http.MethodPost {
		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Parse request
		var req struct {
			Message    string `json:"message"`
			SessionID  string `json:"session_id"`
		}

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
		}

		// Process message through agent
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()

		// Save user message to session if session manager is available
		if s.sessionManager != nil {
			if err := s.sessionManager.AddMessage(ctx, sessionID, "user", req.Message, nil); err != nil {
				log.Printf("Failed to save user message: %v", err)
			}
		}

		response, err := s.agent.ProcessMessage(ctx, req.Message)
		if err != nil {
			log.Printf("Agent error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Save assistant response to session if session manager is available
		if s.sessionManager != nil {
			responseContent := response.TextOrEmpty()
			if err := s.sessionManager.AddMessage(ctx, sessionID, "assistant", responseContent, nil); err != nil {
				log.Printf("Failed to save assistant message: %v", err)
			}
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"session_id": sessionID,
			"content": response.TextOrEmpty(),
		})
		return
	}

	// Default response for unknown paths
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleMemoryAPI(w http.ResponseWriter, r *http.Request) {
	// For GET requests, return memory entries
	if r.Method == http.MethodGet {
		if s.memoryBackend == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  "memory backend not configured",
			})
			return
		}

		// Try to get memory backend interface
		if mem, ok := s.memoryBackend.(interface {
			List(ctx context.Context, category *string) ([]memory.MemoryEntry, error)
		}); ok {
			entries, err := mem.List(r.Context(), nil)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "error",
					"error":  err.Error(),
				})
				return
			}

			// Convert entries to map format
			result := make([]map[string]interface{}, len(entries))
			for i, entry := range entries {
				result[i] = map[string]interface{}{
					"key":        entry.Key,
					"content":     entry.Content,
					"category":    entry.Category,
					"created_at":  entry.CreatedAt,
					"updated_at":  entry.UpdatedAt,
					"metadata":    entry.Metadata,
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "ok",
				"entries": result,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"entries": []map[string]interface{}{},
		})
		return
	}

	// For POST requests, store memory entry
	if r.Method == http.MethodPost {
		if r.Body == nil || r.ContentLength == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  "request body is required",
			})
			return
		}

		if s.memoryBackend == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  "memory backend not configured",
			})
			return
		}

		var req struct {
			Key      string                 `json:"key"`
			Content  string                 `json:"content"`
			Category string                 `json:"category"`
			Metadata map[string]interface{} `json:"metadata,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		// Validate required fields
		if req.Key == "" || req.Content == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"error":  "key and content are required",
			})
			return
		}

		// Try to store using memory backend interface
		if mem, ok := s.memoryBackend.(interface {
			Store(ctx context.Context, key, content string, category *string, metadata map[string]string) error
		}); ok {
			// Convert metadata
			metadata := make(map[string]string)
			for k, v := range req.Metadata {
				metadata[k] = fmt.Sprintf("%v", v)
			}

			category := &req.Category
			if req.Category == "" {
				category = nil
			}

			if err := mem.Store(r.Context(), req.Key, req.Content, category, metadata); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "error",
					"error":  err.Error(),
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok",
				"key":    req.Key,
			})
			return
		}

		// Try using memory.MemoryBackend interface
		if mem, ok := s.memoryBackend.(memory.MemoryBackend); ok {
			// Convert metadata
			metadata := make(map[string]string)
			for k, v := range req.Metadata {
				metadata[k] = fmt.Sprintf("%v", v)
			}

			category := &req.Category
			if req.Category == "" {
				category = nil
			}

			if err := mem.Store(r.Context(), req.Key, req.Content, category, metadata); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "error",
					"error":  err.Error(),
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok",
				"key":    req.Key,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  "memory backend does not support store operation",
		})
		return
	}

	// For other methods, return method not allowed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "error",
		"error":  "method not allowed",
	})
}

func (s *Server) SetAuthMiddleware(middleware func(http.Handler) http.Handler) {
	s.authMiddleware = middleware
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// splitProtocols splits a comma-separated Sec-WebSocket-Protocol header value
func splitProtocols(s string) []string {
	var result []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// formatMemoryEntries formats memory entries for output
func formatMemoryEntries(entries []agent.MemoryEntry) string {
	if len(entries) == 0 {
		return "No memories found matching that query."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d memories:\n", len(entries)))
	for _, entry := range entries {
		category := "general"
		if entry.Category != nil {
			category = *entry.Category
		}
		result.WriteString(fmt.Sprintf("- [%s] %s: %s\n", category, entry.Key, entry.Content))
	}
	return result.String()
}

// handleWebSocket handles WebSocket connections for /ws/chat
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket request: %s %s", r.Method, r.URL.Path)
	
	// Check for protocol header and respond accordingly
	clientProtocols := r.Header.Get("Sec-WebSocket-Protocol")
	responseHeader := http.Header{}
	
	// Extract and validate token from query parameters
	var authToken *auth.Token
	token := r.URL.Query().Get("token")
	if token != "" && s.authService != nil {
		// Try to validate as user token
		if userToken, err := s.authService.ValidateUserToken(token); err == nil {
			authToken = userToken
			log.Printf("WebSocket authenticated as user: %s", userToken.Username)
		} else if adminToken, err := s.authService.ValidateAdminToken(token); err == nil {
			authToken = adminToken
			log.Printf("WebSocket authenticated as admin: %s", adminToken.Username)
		}
	}
	
	// Accept zeroclaw.v1 protocol if offered
	if clientProtocols != "" {
		protocols := splitProtocols(clientProtocols)
		for _, p := range protocols {
			if p == "zeroclaw.v1" {
				responseHeader.Set("Sec-WebSocket-Protocol", "zeroclaw.v1")
				break
			}
		}
	}
	
	// If auth is enabled and no valid token, allow anonymous connection
	// but log a warning
	if s.authService != nil && authToken == nil {
		log.Printf("WebSocket connection allowed: anonymous connection (no token provided)")
		// Don't reject the connection, just allow anonymous access
	}
	
	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	log.Printf("WebSocket upgraded successfully from %s", r.RemoteAddr)

	clientID := fmt.Sprintf("ws_%d", time.Now().UnixNano())
	client := &wsClient{
		conn:     conn,
		sendChan: make(chan []byte, 256),
	}

	s.wsClientsMu.Lock()
	s.wsClients[clientID] = client
	s.wsClientsMu.Unlock()

	defer func() {
		s.wsClientsMu.Lock()
		delete(s.wsClients, clientID)
		s.wsClientsMu.Unlock()
		conn.Close()
	}()

	// Send welcome message
	welcome := map[string]interface{}{
		"type":    "connected",
		"message": "WebSocket connected successfully",
	}
	if authToken != nil {
		welcome["user"] = map[string]interface{}{
			"id":       authToken.UserID,
			"username": authToken.Username,
			"is_admin": authToken.IsAdmin,
		}
	}
	if data, err := json.Marshal(welcome); err == nil {
		conn.WriteMessage(websocket.TextMessage, data)
	}

	// Read messages from client
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		log.Printf("Received WebSocket message: %s", string(message))

		// Parse message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("WebSocket message parse error: %v", err)
			continue
		}

		log.Printf("Parsed message: %+v", msg)

		msgType, _ := msg["type"].(string)
		
		switch msgType {
		case "message", "chat":
			// Handle chat message
			content, _ := msg["content"].(string)
			if content == "" {
				// Try to get content from data field
				if data, ok := msg["data"].(map[string]interface{}); ok {
					content, _ = data["content"].(string)
				}
			}
			
			if content == "" {
				log.Printf("Empty message content")
				continue
			}

			sessionID, _ := msg["session_id"].(string)
			if sessionID == "" {
				sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
			}

			// Call agent
			go s.handleAgentChat(conn, sessionID, content)

		case "ping":
			pong := map[string]interface{}{
				"type":      "pong",
				"timestamp": time.Now().Unix(),
			}
			if data, err := json.Marshal(pong); err == nil {
				conn.WriteMessage(websocket.TextMessage, data)
			}

		default:
			// Echo back for unknown types
			response := map[string]interface{}{
				"type":    "response",
				"message": "Message received",
				"data":    msg,
			}
			if data, err := json.Marshal(response); err == nil {
				conn.WriteMessage(websocket.TextMessage, data)
			}
		}
	}
}

// handleAgentChat processes a chat message through the agent
func (s *Server) handleAgentChat(conn *websocket.Conn, sessionID, content string) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	log.Printf("Processing chat message: %s", content)

	// Save user message to session if session manager is available
	if s.sessionManager != nil {
		if err := s.sessionManager.AddMessage(ctx, sessionID, "user", content, nil); err != nil {
			log.Printf("Failed to save user message: %v", err)
		}
	}

	response, err := s.agent.ProcessMessage(ctx, content)
	if err != nil {
		log.Printf("Agent error: %v", err)
		errMsg := map[string]interface{}{
			"type":      "error",
			"session_id": sessionID,
			"error":      err.Error(),
		}
		if data, err := json.Marshal(errMsg); err == nil {
			conn.WriteMessage(websocket.TextMessage, data)
		}
		return
	}

	// Save assistant response to session if session manager is available
	if s.sessionManager != nil {
		responseContent := response.TextOrEmpty()
		if err := s.sessionManager.AddMessage(ctx, sessionID, "assistant", responseContent, nil); err != nil {
			log.Printf("Failed to save assistant message: %v", err)
		}
	}

	// Send response
	resp := map[string]interface{}{
		"type":          "done",
		"session_id":    sessionID,
		"content":       response.TextOrEmpty(),
		"full_response": response.TextOrEmpty(),
		"finish_reason": "stop",
	}
	if response.Usage != nil {
		resp["usage"] = response.Usage
	}

	if data, err := json.Marshal(resp); err == nil {
		conn.WriteMessage(websocket.TextMessage, data)
		log.Printf("Sent response for session %s", sessionID)
		log.Printf("============end=====================")
	}
}

// maskSensitiveFields masks sensitive fields in config content
func maskSensitiveFields(content string) string {
	// List of sensitive field patterns to mask
	sensitivePatterns := []struct {
		pattern string
		replacement string
	}{
		{`api_key\s*=\s*"[^"]*"`, `api_key = "***MASKED***"`},
		{`api_key\s*=\s*'[^']*'`, `api_key = '***MASKED***'`},
		{`client_secret\s*=\s*"[^"]*"`, `client_secret = "***MASKED***"`},
		{`client_secret\s*=\s*'[^']*'`, `client_secret = '***MASKED***'`},
		{`token\s*=\s*"[^"]*"`, `token = "***MASKED***"`},
		{`token\s*=\s*'[^']*'`, `token = '***MASKED***'`},
		{`secret\s*=\s*"[^"]*"`, `secret = "***MASKED***"`},
		{`secret\s*=\s*'[^']*'`, `secret = '***MASKED***'`},
		{`password\s*=\s*"[^"]*"`, `password = "***MASKED***"`},
		{`password\s*=\s*'[^']*'`, `password = '***MASKED***'`},
		{`api_keys\s*=\s*\[[^\]]*\]`, `api_keys = ["***MASKED***"]`},
	}
	
	result := content
	for _, sp := range sensitivePatterns {
		re := regexp.MustCompile(sp.pattern)
		result = re.ReplaceAllString(result, sp.replacement)
	}
	
	return result
}

// handleWechatLogin handles WeChat login URL generation
func (s *Server) handleWechatLogin(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil || s.authService.WechatClient == nil {
		http.Error(w, "WeChat login not configured", http.StatusInternalServerError)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		state = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	redirectURL := s.authService.WechatClient.GetAuthURL(state)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"login_url": redirectURL,
		"state":     state,
	})
}

// handleWechatCallback handles WeChat login callback
func (s *Server) handleWechatCallback(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil || s.authService.WechatClient == nil || s.userManager == nil {
		http.Error(w, "WeChat login not configured", http.StatusInternalServerError)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	// Get access token
	wechatToken, err := s.authService.WechatClient.GetAccessToken(code)
	if err != nil {
		http.Error(w, "Failed to get access token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user info
	userInfo, err := s.authService.WechatClient.GetUserInfo(wechatToken.AccessToken, wechatToken.OpenID)
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if user exists
	existingUser, err := s.userManager.GetUserByOpenID(userInfo.OpenID)
	if err != nil {
		http.Error(w, "Failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if existingUser == nil {
		// Create new user if not exists
		newUser, err := s.userManager.CreateUser(userInfo.OpenID, userInfo.Nickname, userInfo.HeadImgURL, "")
		if err != nil {
			http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
			return
		}
		existingUser = newUser
	}

	// Check if user is approved
	if existingUser.Status != 1 {
		http.Redirect(w, r, "/#/login/pending", http.StatusFound)
		return
	}

	// Generate token
	authToken, err := s.authService.UserLogin(userInfo.OpenID)
	if err != nil {
		http.Error(w, "Failed to login: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 广播登录成功消息给所有WebSocket客户端
	s.wsClientsMu.RLock()
	defer s.wsClientsMu.RUnlock()
	
	loginMsg := map[string]interface{}{
		"type":  "login.success",
		"token": authToken.Token,
		"user": map[string]interface{}{
			"id":       existingUser.ID,
			"nickname": existingUser.Nickname,
			"avatar":   existingUser.Avatar,
			"status":   existingUser.Status,
		},
	}
	
	msgData, err := json.Marshal(loginMsg)
	if err != nil {
		http.Error(w, "Failed to marshal login message", http.StatusInternalServerError)
		return
	}
	
	for _, client := range s.wsClients {
		// 直接通过WebSocket连接发送消息
		if err := client.conn.WriteMessage(websocket.TextMessage, msgData); err != nil {
			log.Printf("Failed to send WebSocket message: %v", err)
		}
	}
	
	// 返回登录成功响应给手机
	http.Redirect(w, r, "/#/login/success", http.StatusFound)
}

// handleWechatUserInfo handles getting WeChat user info
func (s *Server) handleWechatUserInfo(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// 验证token
	token, err := s.authService.GetTokenInfo(tokenString)
	if err != nil {
		// 尝试验证管理员token
		_, err = s.authService.ValidateAdminToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
	}

	user, err := s.userManager.GetUserByID(token.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"user": map[string]interface{}{
			"id":       user.ID,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"email":    user.Email,
			"status":   user.Status,
			"created_at": user.CreatedAt,
		},
	})
}

// handleAdminLogin handles admin login
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.authService == nil || s.userManager == nil {
		http.Error(w, "Admin login not configured", http.StatusInternalServerError)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 使用AuthService的AdminLogin方法
	token, err := s.authService.AdminLogin(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"token":  token.Token,
		"admin": map[string]interface{}{
			"id":       token.UserID,
			"username": token.Username,
		},
	})
}

// handleAdminUsers handles getting users list for admin
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// 验证管理员token
	_, err := s.authService.ValidateAdminToken(tokenString)
	if err != nil {
		http.Error(w, "Invalid or unauthorized token", http.StatusUnauthorized)
		return
	}

	if s.userManager == nil {
		http.Error(w, "User management not configured", http.StatusInternalServerError)
		return
	}

	// 调用ListUsers方法获取所有用户
	users, err := s.userManager.ListUsers(nil)
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	userList := make([]map[string]interface{}, len(users))
	for i, user := range users {
		userList[i] = map[string]interface{}{
			"id":       user.ID,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"email":    user.Email,
			"status":   user.Status,
			"created_at": user.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"users":  userList,
	})
}

// handleAdminApproveUser handles approving user
func (s *Server) handleAdminApproveUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// 验证管理员token
	_, err := s.authService.ValidateAdminToken(tokenString)
	if err != nil {
		http.Error(w, "Invalid or unauthorized token", http.StatusUnauthorized)
		return
	}

	if s.userManager == nil {
		http.Error(w, "User management not configured", http.StatusInternalServerError)
		return
	}

	var req struct {
		UserID int   `json:"user_id"`
		Status int `json:"status"` // 1: approved, 2: rejected
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 检查用户是否存在
	_, err = s.userManager.GetUserByID(req.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// 调用UpdateUserStatus方法更新用户状态
	err = s.userManager.UpdateUserStatus(req.UserID, req.Status)
	if err != nil {
		http.Error(w, "Failed to update user status", http.StatusInternalServerError)
		return
	}

	// 重新获取更新后的用户信息
	updatedUser, err := s.userManager.GetUserByID(req.UserID)
	if err != nil {
		http.Error(w, "Failed to get updated user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"user": map[string]interface{}{
			"id":       updatedUser.ID,
			"nickname": updatedUser.Nickname,
			"status":   updatedUser.Status,
		},
	})
}

// handleAdminPasswordChange handles admin password change
func (s *Server) handleAdminPasswordChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// 验证管理员token
	adminToken, err := s.authService.ValidateAdminToken(tokenString)
	if err != nil {
		http.Error(w, "Invalid or unauthorized token", http.StatusUnauthorized)
		return
	}

	if s.userManager == nil {
		http.Error(w, "Password change not configured", http.StatusInternalServerError)
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	admin, err := s.userManager.GetAdminByUsername(adminToken.Username)
	if err != nil {
		http.Error(w, "Admin not found", http.StatusNotFound)
		return
	}

	// 调用CheckPasswordHash函数验证旧密码
	if !auth.CheckPasswordHash(req.OldPassword, admin.Password) {
		http.Error(w, "Invalid old password", http.StatusBadRequest)
		return
	}

	// 调用HashPassword函数哈希新密码
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// 调用UpdateAdminPassword方法更新密码
	err = s.userManager.UpdateAdminPassword(admin.ID, hashedPassword)
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Password changed successfully",
	})
}

// handleUserInfo handles getting user info
func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// 验证用户token
	token, err := s.authService.GetTokenInfo(tokenString)
	if err != nil {
		http.Error(w, "Invalid or unauthorized token", http.StatusUnauthorized)
		return
	}

	if s.userManager == nil {
		http.Error(w, "User management not configured", http.StatusInternalServerError)
		return
	}

	user, err := s.userManager.GetUserByID(token.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"user": map[string]interface{}{
			"id":       user.ID,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"email":    user.Email,
			"status":   user.Status,
			"created_at": user.CreatedAt,
		},
	})
}

// handleUserUpdate handles updating user info
func (s *Server) handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// 验证用户token
	token, err := s.authService.GetTokenInfo(tokenString)
	if err != nil {
		http.Error(w, "Invalid or unauthorized token", http.StatusUnauthorized)
		return
	}

	if s.userManager == nil {
		http.Error(w, "User management not configured", http.StatusInternalServerError)
		return
	}

	var req struct {
		Email  string `json:"email"`
		Avatar string `json:"avatar"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := s.userManager.GetUserByID(token.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// 调用UpdateUserInfo方法更新用户信息
	err = s.userManager.UpdateUserInfo(token.UserID, user.Nickname, req.Avatar, req.Email)
	if err != nil {
		http.Error(w, "Failed to update user info", http.StatusInternalServerError)
		return
	}

	// 重新获取更新后的用户信息
	updatedUser, err := s.userManager.GetUserByID(token.UserID)
	if err != nil {
		http.Error(w, "Failed to get updated user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"user": map[string]interface{}{
			"id":       updatedUser.ID,
			"nickname": updatedUser.Nickname,
			"avatar":   updatedUser.Avatar,
			"email":    updatedUser.Email,
			"status":   updatedUser.Status,
		},
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if s.sessionManager == nil {
		http.Error(w, "Session manager not configured", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		limit := 50
		offset := 0
		
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && l == 1 {
				if limit > 100 {
					limit = 100
				}
			}
		}
		
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if o, err := fmt.Sscanf(offsetStr, "%d", &offset); err == nil && o == 1 {
				if offset < 0 {
					offset = 0
				}
			}
		}

		userID := ""
		if s.authService != nil {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				if token, err := s.authService.GetTokenInfo(tokenString); err == nil {
					userID = fmt.Sprintf("%d", token.UserID)
				}
			}
		}

		sessions, err := s.sessionManager.ListSessions(r.Context(), userID, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": sessions,
			"count":    len(sessions),
		})

	case http.MethodPost:
		var req struct {
			Title string `json:"title"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		userID := ""
		if s.authService != nil {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				if token, err := s.authService.GetTokenInfo(tokenString); err == nil {
					userID = fmt.Sprintf("%d", token.UserID)
				}
			}
		}

		newSession, err := s.sessionManager.CreateSession(r.Context(), userID, req.Title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"session": newSession,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	if s.sessionManager == nil {
		http.Error(w, "Session manager not configured", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.Split(path, "/")
	sessionID := parts[0]

	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(parts) > 1 && parts[1] == "messages" {
		s.handleSessionMessages(w, r, sessionID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		session, err := s.sessionManager.GetSession(r.Context(), sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"session": session,
		})

	case http.MethodPut:
		var req struct {
			Title string `json:"title"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := s.sessionManager.UpdateSessionTitle(r.Context(), sessionID, req.Title); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
		})

	case http.MethodDelete:
		if err := s.sessionManager.DeleteSession(r.Context(), sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSessionMessages(w http.ResponseWriter, r *http.Request, sessionID string) {
	switch r.Method {
	case http.MethodGet:
		limit := 100
		offset := 0
		
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && l == 1 {
				if limit > 500 {
					limit = 500
				}
			}
		}
		
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if o, err := fmt.Sscanf(offsetStr, "%d", &offset); err == nil && o == 1 {
				if offset < 0 {
					offset = 0
				}
			}
		}

		messages, err := s.sessionManager.GetMessages(r.Context(), sessionID, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": messages,
			"count":    len(messages),
		})

	case http.MethodPost:
		var req struct {
			Role     string                 `json:"role"`
			Content  string                 `json:"content"`
			Metadata map[string]interface{} `json:"metadata,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Role == "" || req.Content == "" {
			http.Error(w, "Role and content are required", http.StatusBadRequest)
			return
		}

		if err := s.sessionManager.AddMessage(r.Context(), sessionID, req.Role, req.Content, req.Metadata); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
