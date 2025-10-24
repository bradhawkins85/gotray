package main

import "testing"

func TestParseGlobalFlagsIgnoresBuildXWithSeparateValue(t *testing.T) {
	args := []string{"add", "-X", "internal/trmm.embeddedAPIKey=value", "--debug"}
	filtered, debug, offline, importTRMM, err := parseGlobalFlags(args)
	if err != nil {
		t.Fatalf("parseGlobalFlags returned error: %v", err)
	}
	if !debug {
		t.Fatalf("expected debug flag to be enabled")
	}
	if offline {
		t.Fatalf("offline flag should not be set")
	}
	if importTRMM {
		t.Fatalf("importTRMM flag should not be set")
	}
	if len(filtered) != 1 || filtered[0] != "add" {
		t.Fatalf("unexpected filtered args: %#v", filtered)
	}
}

func TestParseGlobalFlagsIgnoresBuildXInline(t *testing.T) {
	args := []string{"add", "-Xinternal/trmm.embeddedAPIKey=value", "-ImportTRMM=true"}
	filtered, debug, offline, importTRMM, err := parseGlobalFlags(args)
	if err != nil {
		t.Fatalf("parseGlobalFlags returned error: %v", err)
	}
	if debug {
		t.Fatalf("debug flag should not be set")
	}
	if offline {
		t.Fatalf("offline flag should not be set")
	}
	if !importTRMM {
		t.Fatalf("importTRMM flag should be set")
	}
	if len(filtered) != 1 || filtered[0] != "add" {
		t.Fatalf("unexpected filtered args: %#v", filtered)
	}
}

func TestParseGlobalFlagsIgnoresQuotedBuildX(t *testing.T) {
	args := []string{"add", "-X\"internal/trmm.embeddedAPIKey=value\""}
	filtered, debug, offline, importTRMM, err := parseGlobalFlags(args)
	if err != nil {
		t.Fatalf("parseGlobalFlags returned error: %v", err)
	}
	if debug || offline || importTRMM {
		t.Fatalf("unexpected flag states: debug=%v offline=%v importTRMM=%v", debug, offline, importTRMM)
	}
	if len(filtered) != 1 || filtered[0] != "add" {
		t.Fatalf("unexpected filtered args: %#v", filtered)
	}
}
