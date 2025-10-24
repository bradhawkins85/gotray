package trmm

import "testing"

func TestDetectOptionsAgentPKFromNumericID(t *testing.T) {
	t.Setenv("TRMM_AGENT_ID", "42")
	opts := DetectOptions()
	if opts.AgentPK != 42 {
		t.Fatalf("expected AgentPK 42, got %d", opts.AgentPK)
	}
	if opts.AgentID != "" {
		t.Fatalf("expected AgentID to be cleared when numeric, got %q", opts.AgentID)
	}
}

func TestDetectOptionsAgentPKEnv(t *testing.T) {
	t.Setenv("TRMM_AGENT_PK", "99")
	opts := DetectOptions()
	if opts.AgentPK != 99 {
		t.Fatalf("expected AgentPK 99, got %d", opts.AgentPK)
	}
}

func TestDetectOptionsAgentIDMatchingAPIKey(t *testing.T) {
        t.Setenv("TRMM_APIKEY", "secret-key")
        t.Setenv("TRMM_AGENT_ID", "secret-key")
        opts := DetectOptions()
        if opts.AgentID != "" {
                t.Fatalf("expected AgentID cleared when matching API key, got %q", opts.AgentID)
        }
}

func TestDetectOptionsAcceptsLegacyAPIKeyCasing(t *testing.T) {
        t.Setenv("TRMM_APIKey", "legacy-secret")
        opts := DetectOptions()
        if opts.APIKey != "legacy-secret" {
                t.Fatalf("expected APIKey from legacy env var, got %q", opts.APIKey)
        }
}

func TestDetectOptionsAgentIDFromRegistryAgentPK(t *testing.T) {
	registry := map[string]string{"AgentPK": "agent-123"}
	opts := detectOptionsWith(registry, func(string) string { return "" })
	if opts.AgentID != "agent-123" {
		t.Fatalf("expected AgentID from registry AgentPK, got %q", opts.AgentID)
	}
}

func TestDetectOptionsEmbeddedAPIKey(t *testing.T) {
	previous := embeddedAPIKey
	embeddedAPIKey = "compiled-secret"
	defer func() { embeddedAPIKey = previous }()

	opts := detectOptionsWith(nil, func(string) string { return "" })
	if opts.APIKey != "compiled-secret" {
		t.Fatalf("expected APIKey from embedded secret, got %q", opts.APIKey)
	}
}

func TestDetectOptionsPrefersEmbeddedAPIKey(t *testing.T) {
	previous := embeddedAPIKey
	embeddedAPIKey = "compiled-secret"
	defer func() { embeddedAPIKey = previous }()

	registry := map[string]string{"APIKey": "registry-secret"}
	env := func(string) string { return "env-secret" }
	opts := detectOptionsWith(registry, env)
	if opts.APIKey != "compiled-secret" {
		t.Fatalf("expected embedded API key to win, got %q", opts.APIKey)
	}
}
