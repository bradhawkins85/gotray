package security

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"

	"github.com/example/gotray/internal/config"
)

const serviceTokenPrefix = "gotray-service|"

// ResolveServiceToken returns the configured IPC token, deriving a stable value
// from the tray secret when no explicit token is provided.
func ResolveServiceToken(secret string) string {
	if compiled := strings.TrimSpace(config.CompiledSecret); compiled != "" {
		return DeriveServiceToken(compiled)
	}

	token := strings.TrimSpace(os.Getenv("GOTRAY_SERVICE_TOKEN"))
	if token != "" {
		return token
	}

	return DeriveServiceToken(secret)
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
