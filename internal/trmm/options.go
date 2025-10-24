package trmm

import (
	"os"
	"strconv"
	"strings"
)

// embeddedAPIKey is populated at build time via -ldflags "-X" when
// the TRMM API key is supplied through CI secrets.
var embeddedAPIKey string

// Options captures configuration for connecting to Tactical RMM.
// Values may be provided via the Windows registry or environment variables.
type Options struct {
	BaseURL  string
	APIKey   string
	AgentID  string
	AgentPK  int
	SiteID   int
	ClientID int
}

// DetectOptions resolves Tactical RMM options from the environment and registry.
func DetectOptions() Options {
	reg := readRegistrySettings()
	return detectOptionsWith(reg, os.Getenv)
}

type envLookup func(string) string

func detectOptionsWith(reg map[string]string, getenv envLookup) Options {
	opts := Options{}
	opts.BaseURL = firstNonEmpty(reg["BaseURL"], getenv("TRMM_BASE_URL"))
	opts.APIKey = firstNonEmpty(embeddedAPIKey, reg["APIKey"], getenv("TRMM_APIKey"), getenv("TRMM_APIKEY"), getenv("TRMM_API_KEY"))
	opts.AgentID = firstNonEmpty(reg["AgentPK"], reg["AgentPk"], reg["AgentID"], getenv("TRMM_AGENT_ID"))
	opts.AgentPK = parseInt(firstNonEmpty(reg["AgentPK"], reg["AgentPk"], getenv("TRMM_AGENT_PK")))
	opts.SiteID = parseInt(firstNonEmpty(reg["SiteID"], getenv("TRMM_SITE_ID")))
	opts.ClientID = parseInt(firstNonEmpty(reg["ClientID"], getenv("TRMM_CLIENT_ID")))

	if opts.AgentID != "" {
		if pk := parseInt(opts.AgentID); pk > 0 {
			opts.AgentPK = pk
			opts.AgentID = ""
		} else if opts.APIKey != "" && strings.EqualFold(strings.TrimSpace(opts.AgentID), strings.TrimSpace(opts.APIKey)) {
			opts.AgentID = ""
		}
	}
	return opts
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseInt(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	out, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0
	}
	return out
}
