package logging

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"unicode/utf8"
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

// LogHTTPRequest emits detailed information about an outbound HTTP request when
// debugging is enabled. Sensitive headers such as API keys are masked prior to
// logging.
func LogHTTPRequest(req *http.Request, body []byte) {
	if !DebugEnabled() || req == nil {
		return
	}

	target := sanitizeURL(req.URL)
	if target == "" {
		target = "<unknown>"
	}

	log.Printf("[DEBUG] HTTP request %s %s", req.Method, target)

	if len(req.Header) > 0 {
		log.Printf("[DEBUG] --> request headers: %s", formatHeaders(req.Header))
	}

	if len(body) > 0 {
		log.Printf("[DEBUG] --> request payload %s", describePayload(body))
	}
}

// LogHTTPResponse emits detailed information about an inbound HTTP response
// when debugging is enabled. Sensitive headers such as API keys are masked
// prior to logging.
func LogHTTPResponse(resp *http.Response, body []byte) {
	if !DebugEnabled() || resp == nil {
		return
	}

	target := "<unknown>"
	if resp.Request != nil {
		target = sanitizeURL(resp.Request.URL)
	}

	log.Printf("[DEBUG] HTTP response %s for %s", resp.Status, target)

	if len(resp.Header) > 0 {
		log.Printf("[DEBUG] <-- response headers: %s", formatHeaders(resp.Header))
	}

	if len(body) > 0 {
		log.Printf("[DEBUG] <-- response payload %s", describePayload(body))
	}
}

func formatHeaders(headers http.Header) string {
	type headerEntry struct {
		name   string
		values []string
	}

	entries := make([]headerEntry, 0, len(headers))
	for name, values := range headers {
		sanitized := make([]string, len(values))
		for idx, value := range values {
			sanitized[idx] = sanitizeSensitiveValue(name, value)
		}
		entries = append(entries, headerEntry{name: name, values: sanitized})
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})

	var b strings.Builder
	for idx, entry := range entries {
		if idx > 0 {
			b.WriteString(", ")
		}
		b.WriteString(entry.name)
		b.WriteString(": [")
		b.WriteString(strings.Join(entry.values, ", "))
		b.WriteString("]")
	}

	return b.String()
}

func describePayload(body []byte) string {
	if utf8.Valid(body) {
		return fmt.Sprintf("(utf-8, %d bytes): %s", len(body), string(body))
	}

	encoded := base64.StdEncoding.EncodeToString(body)
	return fmt.Sprintf("(base64, %d bytes): %s", len(body), encoded)
}

func sanitizeURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	clone := *u

	if clone.RawQuery != "" {
		query := clone.Query()
		sanitized := false
		for key, values := range query {
			if isSensitiveKey(key) {
				sanitized = true
				for idx, value := range values {
					query[key][idx] = sanitizeSensitiveValue(key, value)
				}
			}
		}
		if sanitized {
			clone.RawQuery = query.Encode()
		}
	}

	if clone.User != nil {
		username := clone.User.Username()
		password, hasPassword := clone.User.Password()
		if hasPassword {
			clone.User = url.UserPassword(username, MaskIdentifier(password))
		}
	}

	return clone.String()
}

func isSensitiveKey(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "api-key"),
		strings.Contains(lower, "apikey"),
		strings.Contains(lower, "authorization"),
		strings.Contains(lower, "secret"),
		strings.Contains(lower, "token"):
		return true
	default:
		return false
	}
}

func sanitizeSensitiveValue(name, value string) string {
	if value == "" {
		return value
	}
	if isSensitiveKey(name) {
		return MaskIdentifier(value)
	}
	return value
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
