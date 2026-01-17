package server

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"rtmp_server/internal/logger"

	"github.com/bluenviron/gortmplib"
	"github.com/bluenviron/gortmplib/pkg/codecs"
)

// RTMPServer handles incoming RTMP streams
type RTMPServer struct {
	addr     string
	manager  *Manager
	listener net.Listener
	running  bool
	mu       sync.Mutex
	wg       sync.WaitGroup
}

// NewRTMPServer creates a new RTMP server
func NewRTMPServer(addr string, manager *Manager) *RTMPServer {
	return &RTMPServer{
		addr:    addr,
		manager: manager,
	}
}

// Start starts the RTMP server
func (r *RTMPServer) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil
	}

	listener, err := net.Listen("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("failed to start RTMP server: %w", err)
	}

	r.listener = listener
	r.running = true

	go r.acceptLoop()

	logger.Info("RTMP server started on %s", r.addr)
	return nil
}

// Stop stops the RTMP server
func (r *RTMPServer) Stop() error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	r.running = false
	r.mu.Unlock()

	if r.listener != nil {
		r.listener.Close()
	}

	r.wg.Wait()
	logger.Info("RTMP server stopped")
	return nil
}

// IsRunning returns whether the server is running
func (r *RTMPServer) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Addr returns the server address
func (r *RTMPServer) Addr() string {
	return r.addr
}

func (r *RTMPServer) acceptLoop() {
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			r.mu.Lock()
			running := r.running
			r.mu.Unlock()
			if !running {
				return
			}
			logger.Error("Accept error: %v", err)
			continue
		}

		r.wg.Add(1)
		go r.handleConnection(conn)
	}
}

func (r *RTMPServer) handleConnection(conn net.Conn) {
	defer r.wg.Done()
	defer conn.Close()

	// Panic recovery to prevent server crash
	defer func() {
		if rec := recover(); rec != nil {
			logger.Error("RTMP handler panic: %v", rec)
		}
	}()

	logger.Info("Connection from %s", conn.RemoteAddr())

	// Set initial read deadline for handshake
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Create RTMP server connection
	sc := &gortmplib.ServerConn{
		RW: conn,
	}

	// Initialize connection (handshake)
	err := sc.Initialize()
	if err != nil {
		logger.Error("RTMP handshake failed: %v", err)
		return
	}

	// Accept connection (get intent: publish or play)
	err = sc.Accept()
	if err != nil {
		logger.Error("RTMP accept failed: %v", err)
		return
	}

	if sc.Publish {
		r.handlePublisher(sc, conn)
	} else {
		logger.Warn("Non-publishing connection rejected from %s", conn.RemoteAddr())
	}
}

func (r *RTMPServer) handlePublisher(sc *gortmplib.ServerConn, conn net.Conn) {
	// Extract stream key from URL path
	// URL format: rtmp://host/app/streamkey -> Path = /app/streamkey
	var streamKey string
	if sc.URL != nil {
		streamKey = extractStreamKey(sc.URL.Path)
	} else {
		streamKey = "default"
		logger.Warn("No URL in RTMP connection, using default stream key")
	}

	logger.Info("Publisher connected: %s from %s", streamKey, conn.RemoteAddr())

	// Get or create stream
	stream, err := r.manager.GetOrCreateStream(streamKey)
	if err != nil {
		logger.Error("Failed to create stream: %v", err)
		return
	}

	defer func() {
		r.manager.RemoveStream(streamKey)
		logger.Info("Publisher disconnected: %s", streamKey)
	}()

	// Create reader to receive data
	reader := &gortmplib.Reader{
		Conn: sc,
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = reader.Initialize()
	if err != nil {
		logger.Error("Failed to initialize reader: %v", err)
		return
	}

	tracks := reader.Tracks()
	logger.Info("Stream %s has %d tracks", streamKey, len(tracks))

	// Find H264 and AAC tracks and set up callbacks
	var hasVideo bool

	for _, track := range tracks {
		switch codec := track.Codec.(type) {
		case *codecs.H264:
			hasVideo = true
			logger.Info("Stream %s: H264 video track detected", streamKey)

			// Update muxer with codec parameters if available
			if len(codec.SPS) > 0 && len(codec.PPS) > 0 {
				stream.SetVideoParams(codec.SPS, codec.PPS)
			}

			reader.OnDataH264(track, func(pts time.Duration, dts time.Duration, au [][]byte) {
				stream.WriteH264(pts, dts, au)
			})

		case *codecs.MPEG4Audio:
			logger.Info("Stream %s: AAC audio track detected (SampleRate=%d, Channels=%d)",
				streamKey, codec.Config.SampleRate, codec.Config.ChannelCount)

			// Pass actual audio config to stream for proper HLS muxer setup
			stream.SetAudioParams(codec.Config.SampleRate, codec.Config.ChannelCount)

			reader.OnDataMPEG4Audio(track, func(pts time.Duration, au []byte) {
				stream.WriteAAC(pts, au)
			})
		}
	}

	if !hasVideo {
		logger.Warn("Stream %s: No H264 video track found", streamKey)
	}

	// Start HLS muxer if we have video
	if hasVideo {
		err = stream.StartMuxer()
		if err != nil {
			logger.Error("Failed to start HLS muxer for %s: %v", streamKey, err)
			return
		}
	}

	// Read packets until connection closes
	for {
		r.mu.Lock()
		running := r.running
		r.mu.Unlock()
		if !running {
			break
		}

		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		err = reader.Read()
		if err != nil {
			logger.Info("Stream %s ended: %v", streamKey, err)
			break
		}
	}
}

func extractStreamKey(path string) string {
	// Remove leading slashes
	path = strings.TrimPrefix(path, "/")

	// Common patterns:
	// rtmp://host/live/streamkey -> path after split = "live/streamkey" -> take last part
	// rtmp://host/app/stream -> path after split = "app/stream" -> take last part

	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1] // Return last segment as stream key
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0]
	}
	return "default"
}
