package logging

import (
	"log"
	"strings"
	"sync/atomic"
)

var debugEnabled atomic.Bool

// EnableDebug turns on verbose debug logging for the application lifecycle.
func EnableDebug() {
	debugEnabled.Store(true)
	log.Printf("[DEBUG] debug logging enabled")
}

// DebugEnabled reports whether debug logging is active.
func DebugEnabled() bool {
	return debugEnabled.Load()
}

// Debugf emits a formatted debug log message when debugging is enabled.
func Debugf(format string, args ...interface{}) {
	if !DebugEnabled() {
		return
	}
	log.Printf("[DEBUG] "+format, args...)
}

// MaskIdentifier obscures sensitive identifiers leaving only the last four characters visible.
func MaskIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(trimmed)-4) + trimmed[len(trimmed)-4:]
}
