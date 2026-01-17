package monitor

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Stats contains system monitoring statistics
type Stats struct {
	MemAllocMB    float64
	MemSysMB      float64
	NumGoroutines int
	Uptime        time.Duration
}

// Monitor tracks system resource usage
type Monitor struct {
	mu        sync.Mutex
	stats     Stats
	startTime time.Time
}

var (
	globalMonitor = &Monitor{
		startTime: time.Now(),
	}
)

// Update refreshes the statistics
func (m *Monitor) Update() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Memory stats from Go runtime
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.stats.MemAllocMB = float64(memStats.Alloc) / 1024 / 1024
	m.stats.MemSysMB = float64(memStats.Sys) / 1024 / 1024
	m.stats.NumGoroutines = runtime.NumGoroutine()
	m.stats.Uptime = time.Since(m.startTime)
}

// Get returns current stats
func (m *Monitor) Get() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats
}

// GetStats returns global monitor stats
func GetStats() Stats {
	return globalMonitor.Get()
}

// UpdateStats updates global monitor
func UpdateStats() {
	globalMonitor.Update()
}

// FormatUptime returns human-readable uptime
func FormatUptime(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
