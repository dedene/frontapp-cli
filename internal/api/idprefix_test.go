package api

import (
	"errors"
	"testing"
)

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"conversation ID", "cnv_abc123", "cnv_"},
		{"message ID", "msg_xyz789", "msg_"},
		{"comment ID", "cmt_def456", "cmt_"},
		{"teammate ID", "tea_ghi789", "tea_"},
		{"tag ID", "tag_jkl012", "tag_"},
		{"too short", "ab", ""},
		{"no underscore", "cnvabc123", ""},
		{"underscore too late", "conver_sation", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractPrefix(tt.id); got != tt.want {
				t.Errorf("ExtractPrefix(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestGetResourceType(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"conversation", "cnv_abc123", "conversation"},
		{"message", "msg_xyz789", "message"},
		{"comment", "cmt_def456", "comment"},
		{"teammate", "tea_ghi789", "teammate"},
		{"unknown prefix", "xyz_abc123", ""},
		{"invalid format", "noprefix", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetResourceType(tt.id); got != tt.want {
				t.Errorf("GetResourceType(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestValidateIDPrefix(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		expectedPrefix string
		wantErr        bool
		wantErrType    string
	}{
		{"correct prefix", "cnv_abc123", "cnv_", false, ""},
		{"wrong prefix - msg for conv", "msg_abc123", "cnv_", true, "message"},
		{"unknown prefix", "xyz_abc123", "cnv_", false, ""}, // let API handle it
		{"invalid format", "noprefix", "cnv_", false, ""},   // let API handle it
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIDPrefix(tt.id, tt.expectedPrefix)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIDPrefix(%q, %q) error = %v, wantErr %v",
					tt.id, tt.expectedPrefix, err, tt.wantErr)

				return
			}

			if tt.wantErr {
				var wrongTypeErr *WrongResourceTypeError
				if !errors.As(err, &wrongTypeErr) {
					t.Errorf("expected WrongResourceTypeError, got %T", err)

					return
				}

				if wrongTypeErr.ActualType != tt.wantErrType {
					t.Errorf("WrongResourceTypeError.ActualType = %q, want %q",
						wrongTypeErr.ActualType, tt.wantErrType)
				}
			}
		})
	}
}

func TestWrongResourceTypeError_Error(t *testing.T) {
	err := &WrongResourceTypeError{
		ExpectedType: "conversation",
		ActualType:   "message",
		ID:           "msg_abc123",
	}

	want := "'msg_abc123' is a message ID, but a conversation ID was expected"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
