package logging

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger provides structured logging with level support
type Logger struct {
	level LogLevel
}

// Global logger instance
var defaultLogger *Logger

func init() {
	defaultLogger = NewLogger()
}

// NewLogger creates a new logger with level determined by LOG_LEVEL environment variable
func NewLogger() *Logger {
	level := INFO // default level
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		switch strings.ToUpper(logLevel) {
		case "DEBUG":
			level = DEBUG
		case "INFO":
			level = INFO
		case "WARN", "WARNING":
			level = WARN
		case "ERROR":
			level = ERROR
		}
	}
	return &Logger{level: level}
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.level <= DEBUG
}

// Debug logs debug messages (only if debug is enabled)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= DEBUG {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs info messages
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= INFO {
		log.Printf("[INFO] "+format, args...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= WARN {
		log.Printf("[WARN] "+format, args...)
	}
}

// Error logs error messages (always shown)
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= ERROR {
		log.Printf("[ERROR] "+format, args...)
	}
}

// Package-level convenience functions using the default logger
func IsDebugEnabled() bool {
	return defaultLogger.IsDebugEnabled()
}

func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// generateUUID creates a random UUID for correlation
func generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		rand.Uint32(),
		rand.Uint32()&0xffff,
		rand.Uint32()&0xffff|0x4000,
		rand.Uint32()&0x3fff|0x8000,
		rand.Uint64()&0xffffffffffff)
}

