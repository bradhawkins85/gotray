package trmm

import (
	"os"
	"strconv"
	"strings"
)

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

	opts := Options{}
	opts.BaseURL = firstNonEmpty(reg["BaseURL"], os.Getenv("TRMM_BASE_URL"))
	opts.APIKey = firstNonEmpty(reg["APIKey"], os.Getenv("TRMM_APIKey"), os.Getenv("TRMM_APIKEY"), os.Getenv("TRMM_API_KEY"))
	opts.AgentID = firstNonEmpty(reg["AgentID"], os.Getenv("TRMM_AGENT_ID"))
	opts.AgentPK = parseInt(firstNonEmpty(reg["AgentPK"], reg["AgentPk"], os.Getenv("TRMM_AGENT_PK")))
	opts.SiteID = parseInt(firstNonEmpty(reg["SiteID"], os.Getenv("TRMM_SITE_ID")))
	opts.ClientID = parseInt(firstNonEmpty(reg["ClientID"], os.Getenv("TRMM_CLIENT_ID")))

	if opts.AgentID != "" {
		if pk := parseInt(opts.AgentID); pk > 0 {
			opts.AgentPK = pk
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
