package trmm

import (
	"encoding/base64"
	"strconv"
	"testing"
)

func TestParseMenu(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSize int
	}{
		{
			name:     "direct array",
			input:    `[ {"id":"1","type":"text","label":"hello"} ]`,
			wantSize: 1,
		},
		{
			name:     "wrapped items",
			input:    `{ "items": [ {"id":"1","type":"quit"} ] }`,
			wantSize: 1,
		},
		{
			name:     "wrapped menu",
			input:    `{ "menu": [ {"id":"1","type":"quit"} ] }`,
			wantSize: 1,
		},
		{
			name:     "wrapped menuItems",
			input:    `{ "menuItems": [ {"id":"1","type":"quit"} ] }`,
			wantSize: 1,
		},
		{
			name:     "quoted json",
			input:    strconv.Quote(`{"items":[{"id":"1","type":"quit"}]}`),
			wantSize: 1,
		},
		{
			name:     "base64 encoded json",
			input:    base64.StdEncoding.EncodeToString([]byte(`[ {"id":"1","type":"quit"} ]`)),
			wantSize: 1,
		},
		{
			name:     "empty",
			input:    "",
			wantSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := parseMenu(tt.input)
			if err != nil {
				if tt.wantSize == 0 && tt.input == "" {
					return
				}
				t.Fatalf("parseMenu returned error: %v", err)
			}

			if len(items) != tt.wantSize {
				t.Fatalf("expected %d items, got %d", tt.wantSize, len(items))
			}

			for _, item := range items {
				if item.Type == "" {
					t.Fatalf("expected item type to be populated, got empty value")
				}
			}
		})
	}
}

func TestParseMenuUnsupported(t *testing.T) {
	_, err := parseMenu("not-json")
	if err == nil {
		t.Fatalf("expected error for unsupported payload")
	}
}
