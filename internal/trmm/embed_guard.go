package trmm

import (
	"os"
	"strings"
)

const allowRuntimeAPIKeyEnv = "GOTRAY_ALLOW_RUNTIME_TRMM_APIKEY"

func init() {
	if os.Getenv(allowRuntimeAPIKeyEnv) != "" {
		return
	}

	trimmed := strings.TrimSpace(embeddedAPIKey)
	if trimmed == "" {
		panic("missing Tactical RMM API key: embed TRMM_APIKey at build time with -ldflags \"-X internal/trmm.embeddedAPIKey=<value>\"")
	}
	embeddedAPIKey = trimmed
}
