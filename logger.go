package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger handles file-based logging for the application
type Logger struct {
	file       *os.File
	logger     *log.Logger
	mu         sync.Mutex
	logLevel   LogLevel
	lastLogMsg string
	lastLogTime time.Time
	repeatCount int
}

// NewLogger creates a new file logger
func NewLogger() (*Logger, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create log directory
	logDir := filepath.Join(homeDir, "Library", "Logs", "pager-ops")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create or open log file
	logPath := filepath.Join(logDir, "app.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(file, "", 0)

	l := &Logger{
		file:     file,
		logger:   logger,
		logLevel: INFO, // Default to INFO level
	}

	// Write startup message
	l.writeLog(INFO, "=====================================")
	l.writeLog(INFO, fmt.Sprintf("PagerOps started at %s", time.Now().Format("2006-01-02 15:04:05")))
	l.writeLog(INFO, "=====================================")

	return l, nil
}

// SetLogLevel sets the minimum log level
func (l *Logger) SetLogLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevel = level
}

// writeLog writes a log message with deduplication
func (l *Logger) writeLog(level LogLevel, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Skip if below minimum log level
	if level < l.logLevel {
		return
	}

	// Deduplicate repetitive messages
	now := time.Now()
	if message == l.lastLogMsg && now.Sub(l.lastLogTime) < 5*time.Second {
		l.repeatCount++
		return
	}

	// If we had repeated messages, log the count
	if l.repeatCount > 0 {
		levelStr := l.getLevelString(level)
		timestamp := l.lastLogTime.Format("2006-01-02 15:04:05")
		l.logger.Printf("[%s] %s (repeated %d times)\n", timestamp, levelStr, l.repeatCount)
		l.repeatCount = 0
	}

	// Log the new message
	levelStr := l.getLevelString(level)
	timestamp := now.Format("2006-01-02 15:04:05")
	l.logger.Printf("[%s] %s %s\n", timestamp, levelStr, message)

	l.lastLogMsg = message
	l.lastLogTime = now
}

// getLevelString returns the string representation of a log level
func (l *Logger) getLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "[DEBUG]"
	case INFO:
		return "[INFO ]"
	case WARN:
		return "[WARN ]"
	case ERROR:
		return "[ERROR]"
	default:
		return "[?????]"
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	if l == nil {
		return
	}
	l.writeLog(DEBUG, message)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	if l == nil {
		return
	}
	l.writeLog(INFO, message)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	if l == nil {
		return
	}
	l.writeLog(WARN, message)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	if l == nil {
		return
	}
	l.writeLog(ERROR, message)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Write final repeated count if any
	if l.repeatCount > 0 {
		timestamp := l.lastLogTime.Format("2006-01-02 15:04:05")
		l.logger.Printf("[%s] [INFO ] (repeated %d times)\n", timestamp, l.repeatCount)
	}

	// Write shutdown message
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	l.logger.Printf("[%s] [INFO ] PagerOps shutting down\n", timestamp)
	l.logger.Printf("[%s] [INFO ] =====================================\n", timestamp)

	return l.file.Close()
}

// RotateLogIfNeeded checks if log file is too large and rotates it
func (l *Logger) RotateLogIfNeeded() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Get file info
	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	// If file is larger than 10MB, rotate it
	if info.Size() > 10*1024*1024 {
		// Close current file
		l.file.Close()

		// Get log path
		homeDir, _ := os.UserHomeDir()
		logDir := filepath.Join(homeDir, "Library", "Logs", "pager-ops")
		logPath := filepath.Join(logDir, "app.log")
		oldLogPath := filepath.Join(logDir, fmt.Sprintf("app-%s.log", time.Now().Format("20060102-150405")))

		// Rename current log
		os.Rename(logPath, oldLogPath)

		// Open new log file
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		l.file = file
		l.logger = log.New(file, "", 0)

		// Clean up old logs (keep only last 5)
		l.cleanOldLogs(logDir)
	}

	return nil
}

// cleanOldLogs removes old log files, keeping only the most recent ones
func (l *Logger) cleanOldLogs(logDir string) {
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	var logFiles []os.DirEntry
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".log" && file.Name() != "app.log" {
			logFiles = append(logFiles, file)
		}
	}

	// If we have more than 5 old log files, delete the oldest ones
	if len(logFiles) > 5 {
		// Sort by modification time (oldest first)
		for i := 0; i < len(logFiles)-5; i++ {
			oldFile := filepath.Join(logDir, logFiles[i].Name())
			os.Remove(oldFile)
		}
	}
}