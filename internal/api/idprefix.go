package api

import "strings"

// ResourcePrefixes maps ID prefixes to human-readable resource names.
// Front API IDs follow the pattern: prefix_base36id (e.g., cnv_abc123).
var ResourcePrefixes = map[string]string{
	"cnv_": "conversation",
	"msg_": "message",
	"cmt_": "comment",
	"tea_": "teammate",
	"tag_": "tag",
	"inb_": "inbox",
	"chn_": "channel",
	"ctc_": "contact",
	"acc_": "account",
	"rul_": "rule",
	"lnk_": "link",
	"shf_": "shift",
	"sig_": "signature",
	"evt_": "event",
	"drf_": "draft",
	"top_": "topic",
}

// ExtractPrefix returns the prefix portion of a Front ID (e.g., "cnv_" from "cnv_abc123").
// Returns empty string if no valid prefix found.
func ExtractPrefix(id string) string {
	if len(id) < 4 {
		return ""
	}

	idx := strings.Index(id, "_")
	if idx == -1 || idx > 4 {
		return ""
	}

	return id[:idx+1]
}

// GetResourceType returns the resource type for a given ID based on its prefix.
// Returns empty string if the prefix is not recognized.
func GetResourceType(id string) string {
	prefix := ExtractPrefix(id)
	if prefix == "" {
		return ""
	}

	return ResourcePrefixes[prefix]
}

// ValidateIDPrefix checks if an ID has the expected prefix.
// Returns a WrongResourceTypeError if the ID has a recognized but incorrect prefix.
// Returns nil if the ID has the correct prefix or an unrecognized format.
func ValidateIDPrefix(id, expectedPrefix string) error {
	actualPrefix := ExtractPrefix(id)
	if actualPrefix == "" {
		return nil // Can't validate, let API handle it
	}

	if actualPrefix == expectedPrefix {
		return nil
	}

	// Check if it's a known prefix (wrong type) vs unknown format
	actualType := ResourcePrefixes[actualPrefix]
	if actualType == "" {
		return nil // Unknown prefix, let API handle it
	}

	expectedType := ResourcePrefixes[expectedPrefix]

	return &WrongResourceTypeError{
		ExpectedType: expectedType,
		ActualType:   actualType,
		ID:           id,
	}
}

// GetExpectedPrefixForResource returns the prefix for a resource type name.
func GetExpectedPrefixForResource(resourceType string) string {
	for prefix, rtype := range ResourcePrefixes {
		if rtype == resourceType {
			return prefix
		}
	}

	return ""
}
