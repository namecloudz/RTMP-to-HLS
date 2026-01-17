package server

import (
	"net/http"
	"strings"
	"sync"

	"rtmp_server/internal/logger"
)

// HTTPServer serves HLS content
type HTTPServer struct {
	addr    string
	manager *Manager
	server  *http.Server
	running bool
	mu      sync.Mutex
}

// NewHTTPServer creates a new HTTP server for HLS delivery
func NewHTTPServer(addr string, manager *Manager) *HTTPServer {
	return &HTTPServer{
		addr:    addr,
		manager: manager,
	}
}

// Start starts the HTTP server
func (h *HTTPServer) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return nil
	}

	mux := http.NewServeMux()

	// Handle HLS requests: /live/{streamKey}/...
	mux.HandleFunc("/live/", func(w http.ResponseWriter, r *http.Request) {
		// Parse stream key from path: /live/{streamKey}/index.m3u8 or /live/{streamKey}/segment.ts
		path := strings.TrimPrefix(r.URL.Path, "/live/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 1 {
			http.NotFound(w, r)
			return
		}

		streamKey := parts[0]
		stream := h.manager.GetStream(streamKey)
		if stream == nil || !stream.Active || stream.Muxer == nil {
			http.NotFound(w, r)
			return
		}

		// Let the muxer handle the request
		stream.Muxer.Handle(w, r)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Stream list endpoint
	mux.HandleFunc("/streams", func(w http.ResponseWriter, r *http.Request) {
		streams := h.manager.GetAllStreams()
		w.Header().Set("Content-Type", "text/plain")
		for _, s := range streams {
			w.Write([]byte(s.Key + "\n"))
		}
	})

	h.server = &http.Server{
		Addr:    h.addr,
		Handler: mux,
	}

	go func() {
		logger.Info("HTTP server started on %s", h.addr)
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	h.running = true
	return nil
}

// Stop stops the HTTP server
func (h *HTTPServer) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running || h.server == nil {
		return nil
	}

	err := h.server.Close()
	h.running = false
	logger.Info("HTTP server stopped")
	return err
}

// IsRunning returns whether the server is running
func (h *HTTPServer) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}

// Addr returns the server address
func (h *HTTPServer) Addr() string {
	return h.addr
}
