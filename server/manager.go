package server

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"rtmp_server/internal/logger"

	"github.com/bluenviron/gohlslib"
	"github.com/bluenviron/gohlslib/pkg/codecs"
	"github.com/bluenviron/mediacommon/pkg/codecs/mpeg4audio"
)

// StreamInfo contains information about an active stream
type StreamInfo struct {
	Key       string
	StartTime time.Time
	Bitrate   int64 // bytes per second
	Viewers   int
	Active    bool
}

// Stream represents a single active stream with its HLS muxer
type Stream struct {
	Key       string
	Muxer     *gohlslib.Muxer
	StartTime time.Time
	Active    bool

	// Codec parameters
	sps []byte
	pps []byte

	// Audio config from incoming stream
	audioSampleRate   int
	audioChannelCount int

	// NTP start time for proper HLS timestamps
	ntpStart time.Time

	// Thread-safe state using atomics
	muxerReady atomic.Bool

	// For bitrate calculation (protected by separate lock)
	brateMu    sync.Mutex
	bytesTotal int64
	lastUpdate time.Time
	bitrate    int64
}

// Manager handles multiple concurrent streams
type Manager struct {
	mu      sync.RWMutex
	streams map[string]*Stream
	hlsDir  string
}

// NewManager creates a new stream manager
func NewManager(hlsDir string) *Manager {
	return &Manager{
		streams: make(map[string]*Stream),
		hlsDir:  hlsDir,
	}
}

// GetOrCreateStream returns an existing stream or creates a new one
func (m *Manager) GetOrCreateStream(streamKey string) (*Stream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, exists := m.streams[streamKey]; exists && s.Active {
		return s, nil
	}

	stream := &Stream{
		Key:        streamKey,
		StartTime:  time.Now(),
		Active:     true,
		lastUpdate: time.Now(),
	}

	m.streams[streamKey] = stream
	logger.Info("Stream created: %s", streamKey)
	return stream, nil
}

// RemoveStream removes a stream from the manager
func (m *Manager) RemoveStream(streamKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, exists := m.streams[streamKey]; exists {
		s.Active = false
		if s.Muxer != nil {
			s.Muxer.Close()
		}
		delete(m.streams, streamKey)
		logger.Info("Stream removed: %s", streamKey)
	}
}

// GetStreamInfo returns info about a specific stream
func (m *Manager) GetStreamInfo(streamKey string) *StreamInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if s, exists := m.streams[streamKey]; exists {
		return &StreamInfo{
			Key:       s.Key,
			StartTime: s.StartTime,
			Bitrate:   s.GetBitrate(),
			Active:    s.Active,
		}
	}
	return nil
}

// GetAllStreams returns info about all active streams
func (m *Manager) GetAllStreams() []StreamInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]StreamInfo, 0, len(m.streams))
	for _, s := range m.streams {
		if s.Active {
			result = append(result, StreamInfo{
				Key:       s.Key,
				StartTime: s.StartTime,
				Bitrate:   s.GetBitrate(),
				Active:    s.Active,
			})
		}
	}
	return result
}

// GetStream returns the stream for direct access
func (m *Manager) GetStream(streamKey string) *Stream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.streams[streamKey]
}

// StreamCount returns the number of active streams
func (m *Manager) StreamCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, s := range m.streams {
		if s.Active {
			count++
		}
	}
	return count
}

// SetVideoParams sets the H264 codec parameters
func (s *Stream) SetVideoParams(sps, pps []byte) {
	s.sps = make([]byte, len(sps))
	s.pps = make([]byte, len(pps))
	copy(s.sps, sps)
	copy(s.pps, pps)
}

// SetAudioParams stores the audio configuration from the incoming stream
func (s *Stream) SetAudioParams(sampleRate, channelCount int) {
	s.audioSampleRate = sampleRate
	s.audioChannelCount = channelCount
	logger.Info("Audio config set: SampleRate=%d, Channels=%d", sampleRate, channelCount)
}

// StartMuxer initializes and starts the HLS muxer
func (s *Stream) StartMuxer() error {
	if s.muxerReady.Load() {
		return nil
	}

	// Create HLS muxer with H264 video track
	videoTrack := &gohlslib.Track{
		Codec: &codecs.H264{
			SPS: s.sps,
			PPS: s.pps,
		},
	}

	// Create AAC audio track with actual config from stream, or defaults
	sampleRate := s.audioSampleRate
	channelCount := s.audioChannelCount
	if sampleRate == 0 {
		sampleRate = 48000 // OBS default is 48kHz
		logger.Warn("Using default audio sample rate: 48kHz")
	}
	if channelCount == 0 {
		channelCount = 2 // Stereo
		logger.Warn("Using default audio channels: stereo")
	}

	audioTrack := &gohlslib.Track{
		Codec: &codecs.MPEG4Audio{
			Config: mpeg4audio.AudioSpecificConfig{
				Type:         mpeg4audio.ObjectTypeAACLC,
				SampleRate:   sampleRate,
				ChannelCount: channelCount,
			},
		},
	}

	s.Muxer = &gohlslib.Muxer{
		Variant:         gohlslib.MuxerVariantMPEGTS,
		SegmentCount:    5,
		SegmentDuration: 2 * time.Second,
		VideoTrack:      videoTrack,
		AudioTrack:      audioTrack,
	}

	err := s.Muxer.Start()
	if err != nil {
		return fmt.Errorf("failed to start muxer: %w", err)
	}

	// Set NTP start time for synchronized timestamps
	s.ntpStart = time.Now()
	s.muxerReady.Store(true)
	logger.Info("HLS muxer started for stream: %s", s.Key)
	return nil
}

// WriteH264 writes H264 video data to the muxer
func (s *Stream) WriteH264(pts, dts time.Duration, au [][]byte) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Error("WriteH264 panic: %v", rec)
		}
	}()

	if !s.muxerReady.Load() || s.Muxer == nil {
		return
	}

	// Calculate bytes for bitrate (separate lock)
	var totalBytes int
	for _, nalu := range au {
		totalBytes += len(nalu)
	}
	s.updateBitrate(int64(totalBytes))

	err := s.Muxer.WriteH264(s.ntpStart.Add(pts), pts, au)
	if err != nil {
		// Suppress common DTS discontinuity errors (non-fatal, common with OBS)
		errStr := err.Error()
		if !contains(errStr, "DTS is not monotonically") && !contains(errStr, "unable to extract DTS") {
			logger.Error("Error writing H264: %v", err)
		}
	}
}

// contains is a simple string contains helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// WriteAAC writes AAC audio data to the muxer
func (s *Stream) WriteAAC(pts time.Duration, au []byte) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Error("WriteAAC panic: %v", rec)
		}
	}()

	if !s.muxerReady.Load() || s.Muxer == nil {
		return
	}

	s.updateBitrate(int64(len(au)))

	err := s.Muxer.WriteMPEG4Audio(s.ntpStart.Add(pts), pts, [][]byte{au})
	if err != nil {
		logger.Error("Error writing AAC: %v", err)
	}
}

// updateBitrate updates the bitrate calculation
func (s *Stream) updateBitrate(bytes int64) {
	s.brateMu.Lock()
	defer s.brateMu.Unlock()

	s.bytesTotal += bytes
	now := time.Now()
	elapsed := now.Sub(s.lastUpdate).Seconds()

	if elapsed >= 1.0 {
		s.bitrate = int64(float64(s.bytesTotal) / elapsed)
		s.bytesTotal = 0
		s.lastUpdate = now
	}
}

// GetBitrate returns the current bitrate in bytes per second
func (s *Stream) GetBitrate() int64 {
	s.brateMu.Lock()
	defer s.brateMu.Unlock()
	return s.bitrate
}

// IsMuxerReady returns whether the muxer is ready to serve
func (s *Stream) IsMuxerReady() bool {
	return s.muxerReady.Load()
}

// FormatBitrate returns a human-readable bitrate string
func FormatBitrate(bytesPerSec int64) string {
	kbps := float64(bytesPerSec) * 8 / 1000
	if kbps >= 1000 {
		return fmt.Sprintf("%.1f Mbps", kbps/1000)
	}
	return fmt.Sprintf("%.0f Kbps", kbps)
}

// FormatDuration returns a human-readable duration string
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}
