package security

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

const serviceTokenPrefix = "gotray-service|"

// ResolveServiceToken returns the configured IPC token.
func ResolveServiceToken() string {
	token := strings.TrimSpace(os.Getenv("GOTRAY_SERVICE_TOKEN"))
	if token != "" {
		return token
	}

	return ""
}

// DeriveServiceToken hashes the provided secret into a deterministic token.
func DeriveServiceToken(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(serviceTokenPrefix + secret))
	return hex.EncodeToString(sum[:])
}
