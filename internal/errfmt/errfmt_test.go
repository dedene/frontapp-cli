package errfmt

import (
	"strings"
	"testing"

	"github.com/dedene/frontapp-cli/internal/api"
)

func TestFormat_WrongResourceTypeError(t *testing.T) {
	err := &api.WrongResourceTypeError{
		ExpectedType: "conversation",
		ActualType:   "message",
		ID:           "msg_abc123",
	}

	result := Format(err)

	if !strings.Contains(result, "Wrong ID type") {
		t.Errorf("expected 'Wrong ID type' in result, got: %s", result)
	}

	if !strings.Contains(result, "msg_abc123") {
		t.Errorf("expected ID in result, got: %s", result)
	}

	if !strings.Contains(result, "message ID") {
		t.Errorf("expected 'message ID' in result, got: %s", result)
	}

	if !strings.Contains(result, "conversation ID") {
		t.Errorf("expected 'conversation ID' in result, got: %s", result)
	}

	if !strings.Contains(result, "frontcli messages get msg_abc123") {
		t.Errorf("expected command suggestion in result, got: %s", result)
	}
}

func TestFormat_APIError404WithWrongPrefix(t *testing.T) {
	err := &api.APIError{
		StatusCode:       404,
		Message:          "not found",
		RequestedID:      "msg_abc123",
		ExpectedResource: "conversation",
	}

	result := Format(err)

	if !strings.Contains(result, "Not found (404)") {
		t.Errorf("expected '404' in result, got: %s", result)
	}

	// Should detect wrong prefix and show hint
	if !strings.Contains(result, "msg_abc123") {
		t.Errorf("expected ID in result, got: %s", result)
	}

	if !strings.Contains(result, "message ID") {
		t.Errorf("expected 'message ID' in result, got: %s", result)
	}
}

func TestFormat_APIError404WithCorrectPrefix(t *testing.T) {
	err := &api.APIError{
		StatusCode:       404,
		Message:          "not found",
		RequestedID:      "cnv_abc123",
		ExpectedResource: "conversation",
	}

	result := Format(err)

	if !strings.Contains(result, "Not found (404)") {
		t.Errorf("expected '404' in result, got: %s", result)
	}

	// Correct prefix, so should show generic message
	if !strings.Contains(result, "doesn't exist or you don't have access") {
		t.Errorf("expected generic 404 message, got: %s", result)
	}
}

func TestGetSuggestionForResource(t *testing.T) {
	tests := []struct {
		resourceType string
		id           string
		wantContains string
	}{
		{"conversation", "cnv_abc", "conv get cnv_abc"},
		{"message", "msg_abc", "messages get msg_abc"},
		{"comment", "cmt_abc", "comments get cmt_abc"},
		{"contact", "ctc_abc", "contacts get ctc_abc"},
		{"unknown", "xxx_abc", ""},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			got := getSuggestionForResource(tt.resourceType, tt.id)

			if tt.wantContains == "" {
				if got != "" {
					t.Errorf("expected empty suggestion, got: %s", got)
				}
				return
			}

			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("getSuggestionForResource(%q, %q) = %q, want contains %q",
					tt.resourceType, tt.id, got, tt.wantContains)
			}
		})
	}
}
