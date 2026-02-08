package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"codex-agent-team/internal/session"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"nhooyr.io/websocket"
)

// Server wraps the HTTP API and WebSocket hub.
type Server struct {
	router       *chi.Mux
	sessionMgr   *session.Manager
	codexBin     string
	repoPath     string
	hub          *Hub
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
}

// NewServer creates a new API server.
func NewServer(codexBin, repoPath string) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		codexBin:   codexBin,
		repoPath:   repoPath,
		sessionMgr: session.NewManager(codexBin, repoPath),
		hub:        NewHub(),
		shutdownCh: make(chan struct{}),
	}

	s.setupMiddleware()
	s.setupRoutes()

	// Start the hub broadcast loop
	go s.hub.Run()

	return s
}

// setupMiddleware configures server middleware.
func (s *Server) setupMiddleware() {
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	s.router.Get("/", s.handleIndex)

	// Session API
	s.router.Post("/api/sessions", s.handleCreateSession)
	s.router.Get("/api/sessions/{id}", s.handleGetSession)
	s.router.Post("/api/sessions/{id}/decompose", s.handleDecompose)
	s.router.Post("/api/sessions/{id}/execute", s.handleExecute)
	s.router.Post("/api/sessions/{id}/merge", s.handleMerge)
	s.router.Get("/api/sessions/{id}/tasks", s.handleGetTasks)

	// WebSocket endpoint
	s.router.Get("/ws/sessions/{id}", s.handleWebSocket)
}

// handleIndex serves the API index.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version": "1.0.0",
		"name":    "Codex Agent Team API",
	})
}

// handleCreateSession creates a new session.
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserTask string `json:"userTask"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	sess, err := s.sessionMgr.Create(ctx, req.UserTask)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast(sess.ID, Event{
		Type: "session.created",
		Data: sess,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// handleGetSession retrieves a session by ID.
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, ok := s.sessionMgr.Get(id)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// handleDecompose triggers task decomposition.
func (s *Server) handleDecompose(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, ok := s.sessionMgr.Get(id)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if err := sess.Decompose(ctx); err != nil {
		s.hub.Broadcast(id, Event{
			Type: "session.error",
			Data: map[string]string{"error": err.Error()},
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast updated tasks
	tasks := sess.DAG.GetTasks()
	s.hub.Broadcast(id, Event{
		Type: "session.decomposed",
		Data: map[string]any{"tasks": tasks},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "decomposed"})
}

// handleExecute starts task execution.
func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, ok := s.sessionMgr.Get(id)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if err := sess.Execute(ctx); err != nil {
		s.hub.Broadcast(id, Event{
			Type: "session.error",
			Data: map[string]string{"error": err.Error()},
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast(id, Event{
		Type: "session.executing",
		Data: map[string]string{"status": "running"},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "executing"})
}

// handleMerge triggers merging of completed tasks.
func (s *Server) handleMerge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, ok := s.sessionMgr.Get(id)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if err := sess.Merge(ctx); err != nil {
		s.hub.Broadcast(id, Event{
			Type: "session.error",
			Data: map[string]string{"error": err.Error()},
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast(id, Event{
		Type: "session.merged",
		Data: map[string]string{"status": "completed"},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "merged"})
}

// handleGetTasks returns all tasks in a session.
func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, ok := s.sessionMgr.Get(id)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	tasks := sess.DAG.GetTasks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// handleWebSocket handles WebSocket connections for real-time updates.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	// Check if session exists
	if _, ok := s.sessionMgr.Get(sessionID); !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	opts := &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:5173", "localhost:3000"},
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		return
	}

	client := NewClient(sessionID, conn, s.hub)
	s.hub.Register(client)

	// Start client read/write loops
	go client.ReadLoop()
	go client.WriteLoop()
}

// Start starts the HTTP server.
func (s *Server) Start(addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Shutdown handler
	go func() {
		<-s.shutdownCh
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	return server.ListenAndServe()
}

// Shutdown stops the server gracefully.
func (s *Server) Shutdown() {
	s.shutdownOnce.Do(func() {
		close(s.shutdownCh)
	})
}
