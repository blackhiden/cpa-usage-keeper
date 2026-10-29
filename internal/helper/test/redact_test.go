package test

import (
	"testing"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/helper"
)

func TestRedactSensitiveValueUsesCanonicalFormat(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"sk-BabcdefghijklmnopqrstuvwxyzmaWyTA", "sk-*********maWyTA"},
		{"short", "*********"},
		{"sk-123456", "*********"},
		{"", "unknown"},
		{"unknown", "unknown"},
	} {
		t.Run(test.input, func(t *testing.T) {
			if got := helper.RedactSensitiveValue(test.input); got != test.want {
				t.Fatalf("masked value = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCPAAPIKeyDisplayNamePrefersKeyName(t *testing.T) {
	row := entities.CPAAPIKey{
		APIKey:     "sk-alpha123456",
		KeyName:    "  CPA Key Name  ",
		KeyAlias:   "User Alias",
		DisplayKey: "sk-B********************************Zejy",
	}

	if got := helper.CPAAPIKeyDisplayName(row); got != "CPA Key Name" {
		t.Fatalf("expected KeyName to take priority, got %q", got)
	}
}

func TestCPAAPIKeyDisplayNameFallsBackToAlias(t *testing.T) {
	row := entities.CPAAPIKey{
		APIKey:     "sk-alpha123456",
		KeyAlias:   "  Production  ",
		DisplayKey: "sk-B********************************Zejy",
	}

	if got := helper.CPAAPIKeyDisplayName(row); got != "Production" {
		t.Fatalf("expected alias label, got %q", got)
	}
}

func TestCPAAPIKeyDisplayNameFallsBackToMaskedRawKey(t *testing.T) {
	row := entities.CPAAPIKey{APIKey: "sk-alpha123456", DisplayKey: "sk-B********************************Zejy"}

	if got := helper.CPAAPIKeyDisplayName(row); got != "sk-*********123456" {
		t.Fatalf("expected canonical masked key fallback, got %q", got)
	}
}

func TestCPAAPIKeyDisplayNameThreeTierPriority(t *testing.T) {
	tests := []struct {
		name    string
		row     entities.CPAAPIKey
		want    string
		message string
	}{
		{
			name: "all_present_prefers_keyname",
			row: entities.CPAAPIKey{
				APIKey:   "sk-alpha123456",
				KeyName:  "CPA Name",
				KeyAlias: "User Alias",
			},
			want:    "CPA Name",
			message: "should prefer KeyName when all are present",
		},
		{
			name: "keyname_empty_uses_alias",
			row: entities.CPAAPIKey{
				APIKey:   "sk-alpha123456",
				KeyName:  "",
				KeyAlias: "User Alias",
			},
			want:    "User Alias",
			message: "should use KeyAlias when KeyName is empty",
		},
		{
			name: "both_empty_uses_masked",
			row: entities.CPAAPIKey{
				APIKey:   "sk-alpha123456",
				KeyName:  "",
				KeyAlias: "",
			},
			want:    "sk-*********123456",
			message: "should use masked key when both KeyName and KeyAlias are empty",
		},
		{
			name: "whitespace_keyname_uses_alias",
			row: entities.CPAAPIKey{
				APIKey:   "sk-alpha123456",
				KeyName:  "   ",
				KeyAlias: "User Alias",
			},
			want:    "User Alias",
			message: "should treat whitespace-only KeyName as empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helper.CPAAPIKeyDisplayName(tt.row); got != tt.want {
				t.Fatalf("%s: got %q, want %q", tt.message, got, tt.want)
			}
		})
	}
}

func TestCPAAPIKeyMaskedDisplayKeyMasksRawKeyWithCanonicalFormat(t *testing.T) {
	row := entities.CPAAPIKey{APIKey: "sk-BabcdefghijklmnopqrstuvwxyzmaWyTA", DisplayKey: "sk-B********************************maWy"}

	if got := helper.CPAAPIKeyMaskedDisplayKey(row); got != "sk-*********maWyTA" {
		t.Fatalf("expected canonical display key, got %q", got)
	}
}

func TestCPAAPIKeyMaskedDisplayKeyFallsBackToStoredDisplayKeyWhenRawKeyIsMissing(t *testing.T) {
	row := entities.CPAAPIKey{DisplayKey: "sk-*********maWyTA"}

	if got := helper.CPAAPIKeyMaskedDisplayKey(row); got != "sk-*********maWyTA" {
		t.Fatalf("expected stored display key fallback, got %q", got)
	}
}
