package server

import (
	"crypto/tls"
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
	useSSL  bool
}

// NewHTTPServer creates a new HTTP server for HLS delivery
func NewHTTPServer(addr string, manager *Manager) *HTTPServer {
	return &HTTPServer{
		addr:    addr,
		manager: manager,
	}
}

// createMux creates and returns the HTTP router/mux
func (h *HTTPServer) createMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Handle HLS requests: /live/{streamKey}/...
	mux.HandleFunc("/live/", func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers for cross-origin playback
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

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

	// Stream list endpoint (JSON)
	mux.HandleFunc("/api/streams", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		streams := h.manager.GetAllStreams()
		if len(streams) == 0 {
			w.Write([]byte("[]"))
			return
		}
		// Simple JSON output
		w.Write([]byte("["))
		for i, s := range streams {
			if i > 0 {
				w.Write([]byte(","))
			}
			w.Write([]byte(`{"key":"` + s.Key + `","bitrate":` + formatInt(s.Bitrate) + `}`))
		}
		w.Write([]byte("]"))
	})

	// Stream list endpoint (text, legacy)
	mux.HandleFunc("/streams", func(w http.ResponseWriter, r *http.Request) {
		streams := h.manager.GetAllStreams()
		w.Header().Set("Content-Type", "text/plain")
		for _, s := range streams {
			w.Write([]byte(s.Key + "\n"))
		}
	})

	return mux
}

// formatInt converts int64 to string
func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	if negative {
		result = append([]byte{'-'}, result...)
	}
	return string(result)
}

// Start starts the HTTP server (no SSL)
func (h *HTTPServer) Start() error {
	return h.startServer("", "")
}

// StartWithTLS starts the HTTP server with TLS/SSL
func (h *HTTPServer) StartWithTLS(certFile, keyFile string) error {
	return h.startServer(certFile, keyFile)
}

// startServer starts the server, optionally with TLS
func (h *HTTPServer) startServer(certFile, keyFile string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return nil
	}

	mux := h.createMux()
	h.useSSL = certFile != "" && keyFile != ""

	h.server = &http.Server{
		Addr:    h.addr,
		Handler: mux,
	}

	// If TLS, configure it
	if h.useSSL {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			},
		}
		h.server.TLSConfig = tlsConfig
	}

	go func() {
		var err error
		if h.useSSL {
			logger.Info("🔒 HTTPS server started on %s (SSL enabled)", h.addr)
			err = h.server.ListenAndServeTLS(certFile, keyFile)
		} else {
			logger.Info("HTTP server started on %s", h.addr)
			err = h.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
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
	if h.useSSL {
		logger.Info("HTTPS server stopped")
	} else {
		logger.Info("HTTP server stopped")
	}
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

// IsSSL returns whether SSL is enabled
func (h *HTTPServer) IsSSL() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.useSSL
}
