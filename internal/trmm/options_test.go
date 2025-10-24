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
	t.Setenv("TRMM_APIKey", "secret-key")
	t.Setenv("TRMM_AGENT_ID", "secret-key")
	opts := DetectOptions()
	if opts.AgentID != "" {
		t.Fatalf("expected AgentID cleared when matching API key, got %q", opts.AgentID)
	}
}
