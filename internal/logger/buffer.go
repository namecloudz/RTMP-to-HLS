package logger

import (
	"fmt"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel int

const (
	LevelInfo LogLevel = iota
	LevelWarn
	LevelError
)

func (l LogLevel) String() string {
	switch l {
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Entry represents a single log entry
type Entry struct {
	Time    time.Time
	Level   LogLevel
	Message string
}

// Buffer is a thread-safe circular log buffer
type Buffer struct {
	mu      sync.Mutex
	entries []Entry
	maxSize int
}

// Global buffer instance
var (
	globalBuffer = &Buffer{
		entries: make([]Entry, 0, 500),
		maxSize: 500,
	}
)

// Add adds a new log entry to the buffer
func (b *Buffer) Add(level LogLevel, format string, args ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry := Entry{
		Time:    time.Now(),
		Level:   level,
		Message: fmt.Sprintf(format, args...),
	}

	if len(b.entries) >= b.maxSize {
		// Remove oldest entry
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, entry)
}

// GetEntries returns a copy of all log entries
func (b *Buffer) GetEntries() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]Entry, len(b.entries))
	copy(result, b.entries)
	return result
}

// Clear removes all log entries
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = b.entries[:0]
}

// Global convenience functions
func Info(format string, args ...interface{}) {
	globalBuffer.Add(LevelInfo, format, args...)
}

func Warn(format string, args ...interface{}) {
	globalBuffer.Add(LevelWarn, format, args...)
}

func Error(format string, args ...interface{}) {
	globalBuffer.Add(LevelError, format, args...)
}

func GetLogs() []Entry {
	return globalBuffer.GetEntries()
}

func ClearLogs() {
	globalBuffer.Clear()
}
