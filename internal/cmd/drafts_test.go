package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"

	"github.com/dedene/frontapp-cli/internal/api"
)

func TestDraftCreateAcceptsStringVersionResponse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var (
		gotPath      string
		gotPatchPath string
		gotBody      map[string]any
		gotPatchBody map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/channels/cha_v4x/drafts":
			gotPath = r.URL.Path

			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"type":"email",
				"is_inbound":false,
				"draft_mode":"shared",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"hello world",
				"text":"hello world",
				"subject":"Test draft",
				"recipients":[{"handle":"alice@example.com","role":"to"}],
				"_links":{"related":{"conversation":"https://api2.frontapp.com/conversations/cnv_456"}}
			}`)
		case "/conversations/cnv_456":
			gotPatchPath = r.URL.Path

			if err := json.NewDecoder(r.Body).Decode(&gotPatchBody); err != nil {
				t.Fatalf("decode patch body: %v", err)
			}

			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	restoreStdout := captureFile(t, &os.Stdout)

	cmd := DraftCreateCmd{
		Channel: "cha_v4x",
		To:      "alice@example.com",
		Subject: "Test draft",
		Body:    "hello world",
		Author:  "tea_author",
		Inbox:   "inb_team",
	}
	flags := &RootFlags{JSON: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	stdout := restoreStdout()

	if gotPath != "/channels/cha_v4x/drafts" {
		t.Fatalf("expected path /channels/cha_v4x/drafts, got %s", gotPath)
	}

	if gotBody["body"] != "hello world" {
		t.Fatalf("expected body to be sent, got %#v", gotBody)
	}

	if gotBody["subject"] != "Test draft" {
		t.Fatalf("expected subject to be sent, got %#v", gotBody)
	}

	if gotBody["author_id"] != "tea_author" {
		t.Fatalf("expected author_id to be sent, got %#v", gotBody["author_id"])
	}

	to, ok := gotBody["to"].([]any)
	if !ok || len(to) != 1 || to[0] != "alice@example.com" {
		t.Fatalf("expected to recipient to be sent, got %#v", gotBody["to"])
	}

	if gotPatchPath != "/conversations/cnv_456" {
		t.Fatalf("expected conversation patch path, got %s", gotPatchPath)
	}

	if gotPatchBody["inbox_id"] != "inb_team" {
		t.Fatalf("expected inbox_id in patch body, got %#v", gotPatchBody["inbox_id"])
	}

	if !strings.Contains(stdout, `"version": "draft-ver-123"`) {
		t.Fatalf("expected json output to include string version, got %q", stdout)
	}
}

func TestDraftCreateRequiresAuthorAndInbox(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	flags := &RootFlags{}

	err := (&DraftCreateCmd{
		Channel: "cha_v4x",
		To:      "alice@example.com",
		Body:    "hello world",
		Inbox:   "inb_123",
	}).Run(flags)
	if err == nil || !strings.Contains(err.Error(), "--author is required") {
		t.Fatalf("expected missing author validation, got %v", err)
	}

	err = (&DraftCreateCmd{
		Channel: "cha_v4x",
		To:      "alice@example.com",
		Body:    "hello world",
		Author:  "tea_123",
	}).Run(flags)
	if err == nil || !strings.Contains(err.Error(), "--inbox is required") {
		t.Fatalf("expected missing inbox validation, got %v", err)
	}
}

func TestDraftCreateAssignsAuthorAndPatchesConversationDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var (
		gotPostPath  string
		gotPatchPath string
		gotPostBody  map[string]any
		gotPatchBody map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/channels/cha_v4x/drafts":
			gotPostPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&gotPostBody); err != nil {
				t.Fatalf("decode post body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"type":"email",
				"is_inbound":false,
				"draft_mode":"private",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"hello world",
				"text":"hello world",
				"subject":"Test draft",
				"_links":{"related":{"conversation":"https://api2.frontapp.com/conversations/cnv_456"}}
			}`)
		case "/conversations/cnv_456":
			gotPatchPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&gotPatchBody); err != nil {
				t.Fatalf("decode patch body: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	restoreStdout := captureFile(t, &os.Stdout)

	cmd := DraftCreateCmd{
		Channel: "cha_v4x",
		To:      "alice@example.com",
		Subject: "Test draft",
		Body:    "hello world",
		Author:  "tea_author",
		Inbox:   "inb_123",
	}
	flags := &RootFlags{Plain: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	stdout := restoreStdout()

	if gotPostPath != "/channels/cha_v4x/drafts" {
		t.Fatalf("expected draft post path, got %s", gotPostPath)
	}

	if gotPostBody["author_id"] != "tea_author" {
		t.Fatalf("expected author_id in create request, got %#v", gotPostBody)
	}

	if gotPostBody["mode"] != "private" {
		t.Fatalf("expected default private mode, got %#v", gotPostBody["mode"])
	}

	if gotPatchPath != "/conversations/cnv_456" {
		t.Fatalf("expected conversation patch path, got %s", gotPatchPath)
	}

	if gotPatchBody["inbox_id"] != "inb_123" {
		t.Fatalf("expected inbox_id in patch body, got %#v", gotPatchBody)
	}

	if gotPatchBody["assignee_id"] != "tea_author" {
		t.Fatalf("expected assignee to default to author, got %#v", gotPatchBody)
	}

	if !strings.Contains(stdout, "Draft created: msg_draft_123") {
		t.Fatalf("expected create output, got %q", stdout)
	}
}

func TestDraftCreateUsesExplicitAssigneeAndMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotPostBody map[string]any
	var gotPatchBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/channels/cha_v4x/drafts":
			if err := json.NewDecoder(r.Body).Decode(&gotPostBody); err != nil {
				t.Fatalf("decode post body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"type":"email",
				"is_inbound":false,
				"draft_mode":"shared",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"hello world",
				"text":"hello world",
				"_links":{"related":{"conversation":"https://api2.frontapp.com/conversations/cnv_456"}}
			}`)
		case "/conversations/cnv_456":
			if err := json.NewDecoder(r.Body).Decode(&gotPatchBody); err != nil {
				t.Fatalf("decode patch body: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftCreateCmd{
		Channel:  "cha_v4x",
		To:       "alice@example.com",
		Body:     "hello world",
		Author:   "tea_author",
		Inbox:    "inb_123",
		Assignee: "tea_assignee",
		Mode:     "shared",
	}
	flags := &RootFlags{JSON: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if gotPostBody["mode"] != "shared" {
		t.Fatalf("expected shared mode in create request, got %#v", gotPostBody["mode"])
	}

	if gotPatchBody["assignee_id"] != "tea_assignee" {
		t.Fatalf("expected explicit assignee in patch body, got %#v", gotPatchBody["assignee_id"])
	}
}

func TestDraftCreateAddsDefaultSignatureByDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotPostBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/channels/cha_v4x/drafts":
			if err := json.NewDecoder(r.Body).Decode(&gotPostBody); err != nil {
				t.Fatalf("decode post body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"draft_mode":"private",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"hello world",
				"text":"hello world",
				"_links":{"related":{"conversation":"https://api2.frontapp.com/conversations/cnv_456"}}
			}`)
		case "/conversations/cnv_456":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftCreateCmd{
		Channel:          "cha_v4x",
		Body:             "hello world",
		Author:           "tea_author",
		Inbox:            "inb_123",
		DefaultSignature: true,
	}
	flags := &RootFlags{Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	value, ok := gotPostBody["should_add_default_signature"].(bool)
	if !ok || !value {
		t.Fatalf("expected should_add_default_signature=true, got %#v", gotPostBody["should_add_default_signature"])
	}

	if _, exists := gotPostBody["signature_id"]; exists {
		t.Fatalf("did not expect signature_id when using default signature, got %#v", gotPostBody["signature_id"])
	}
}

func TestDraftCreateOmitsDefaultSignatureWhenDisabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotPostBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/channels/cha_v4x/drafts":
			if err := json.NewDecoder(r.Body).Decode(&gotPostBody); err != nil {
				t.Fatalf("decode post body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"draft_mode":"private",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"hello world",
				"text":"hello world",
				"_links":{"related":{"conversation":"https://api2.frontapp.com/conversations/cnv_456"}}
			}`)
		case "/conversations/cnv_456":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftCreateCmd{
		Channel:          "cha_v4x",
		Body:             "hello world",
		Author:           "tea_author",
		Inbox:            "inb_123",
		DefaultSignature: false,
	}
	flags := &RootFlags{Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, exists := gotPostBody["should_add_default_signature"]; exists {
		t.Fatalf("did not expect should_add_default_signature when disabled, got %#v", gotPostBody["should_add_default_signature"])
	}

	if _, exists := gotPostBody["signature_id"]; exists {
		t.Fatalf("did not expect signature_id when default signature disabled, got %#v", gotPostBody["signature_id"])
	}
}

func TestDraftCreateUsesExplicitSignatureID(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotPostBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/channels/cha_v4x/drafts":
			if err := json.NewDecoder(r.Body).Decode(&gotPostBody); err != nil {
				t.Fatalf("decode post body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"draft_mode":"private",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"hello world",
				"text":"hello world",
				"_links":{"related":{"conversation":"https://api2.frontapp.com/conversations/cnv_456"}}
			}`)
		case "/conversations/cnv_456":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftCreateCmd{
		Channel:   "cha_v4x",
		Body:      "hello world",
		Author:    "tea_author",
		Inbox:     "inb_123",
		Signature: "sig_123",
	}
	flags := &RootFlags{Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if gotPostBody["signature_id"] != "sig_123" {
		t.Fatalf("expected signature_id, got %#v", gotPostBody["signature_id"])
	}

	if _, exists := gotPostBody["should_add_default_signature"]; exists {
		t.Fatalf("did not expect should_add_default_signature with explicit signature, got %#v", gotPostBody["should_add_default_signature"])
	}
}

func TestDraftListRendersStringVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations/cnv_123/drafts" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"_results":[
				{
					"id":"msg_draft_123",
					"type":"email",
					"is_inbound":false,
					"draft_mode":"shared",
					"version":"draft-ver-123",
					"created_at":1710000000,
					"body":"hello world",
					"text":"hello world",
					"subject":"Test draft"
				}
			]
		}`)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	restoreStdout := captureFile(t, &os.Stdout)

	cmd := DraftListCmd{ConvID: "cnv_123"}
	flags := &RootFlags{Plain: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	stdout := restoreStdout()

	if !strings.Contains(stdout, "draft-ver-123") {
		t.Fatalf("expected plain output to include string version, got %q", stdout)
	}
}

func TestDraftGetUsesMessageResourceAndRendersStringVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path != "/messages/msg_draft_123" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"id":"msg_draft_123",
			"type":"email",
			"is_inbound":false,
			"draft_mode":"shared",
			"version":"draft-ver-123",
			"created_at":1710000000,
			"body":"hello world",
			"text":"hello world",
			"subject":"Test draft"
		}`)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	restoreStdout := captureFile(t, &os.Stdout)

	cmd := DraftGetCmd{ID: "msg_draft_123"}
	flags := &RootFlags{Plain: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	stdout := restoreStdout()

	if gotPath != "/messages/msg_draft_123" {
		t.Fatalf("expected message resource path, got %s", gotPath)
	}

	if !strings.Contains(stdout, "Version: draft-ver-123") {
		t.Fatalf("expected human output to print string version, got %q", stdout)
	}
}

func TestDraftGetRejectsNonDraftMessage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages/msg_not_draft" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"id":"msg_not_draft",
			"type":"email",
			"is_inbound":false,
			"created_at":1710000000,
			"body":"hello world",
			"text":"hello world",
			"subject":"Regular message"
		}`)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftGetCmd{ID: "msg_not_draft"}
	flags := &RootFlags{Plain: true, Account: "test@example.com"}

	err := cmd.Run(flags)
	if err == nil {
		t.Fatal("expected non-draft message to be rejected")
	}

	if !strings.Contains(err.Error(), "not a draft") {
		t.Fatalf("expected not-a-draft error, got %v", err)
	}
}

func TestDraftUpdateSendsStringVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"id":"msg_draft_123",
			"type":"email",
			"is_inbound":false,
			"draft_mode":"shared",
			"version":"draft-ver-456",
			"created_at":1710000000,
			"body":"updated body",
			"text":"updated body",
			"subject":"Updated draft"
		}`)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftUpdateCmd{
		ID:           "msg_draft_123",
		DraftVersion: "draft-ver-123",
		Body:         "updated body",
		Subject:      "Updated draft",
	}
	flags := &RootFlags{JSON: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if gotPath != "/drafts/msg_draft_123" {
		t.Fatalf("expected path /drafts/msg_draft_123, got %s", gotPath)
	}

	version, ok := gotBody["version"].(string)
	if !ok || version != "draft-ver-123" {
		t.Fatalf("expected string version in request body, got %#v", gotBody["version"])
	}
}

func TestDraftUpdateSendsChannelIDWhenProvided(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"id":"msg_draft_123",
			"type":"email",
			"is_inbound":false,
			"draft_mode":"shared",
			"version":"draft-ver-456",
			"created_at":1710000000,
			"body":"updated body",
			"text":"updated body",
			"subject":"Updated draft"
		}`)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftUpdateCmd{
		ID:           "msg_draft_123",
		Channel:      "cha_v4x",
		DraftVersion: "draft-ver-123",
		Body:         "updated body",
	}
	flags := &RootFlags{JSON: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	channelID, ok := gotBody["channel_id"].(string)
	if !ok || channelID != "cha_v4x" {
		t.Fatalf("expected channel_id in request body, got %#v", gotBody["channel_id"])
	}
}

func TestDraftUpdateHumanOutputUsesStringVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"id":"msg_draft_123",
			"type":"email",
			"is_inbound":false,
			"draft_mode":"shared",
			"version":"draft-ver-456",
			"created_at":1710000000,
			"body":"updated body",
			"text":"updated body",
			"subject":"Updated draft"
		}`)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	restoreStdout := captureFile(t, &os.Stdout)

	cmd := DraftUpdateCmd{
		ID:           "msg_draft_123",
		DraftVersion: "draft-ver-123",
		Body:         "updated body",
	}
	flags := &RootFlags{Plain: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	stdout := restoreStdout()

	if !strings.Contains(stdout, "Draft updated (new version: draft-ver-456)") {
		t.Fatalf("expected human output to print string version, got %q", stdout)
	}
}

func TestDraftDeleteSendsStringVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftDeleteCmd{
		ID:           "msg_draft_123",
		DraftVersion: "draft-ver-123",
	}
	flags := &RootFlags{Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if gotPath != "/drafts/msg_draft_123" {
		t.Fatalf("expected path /drafts/msg_draft_123, got %s", gotPath)
	}

	version, ok := gotBody["version"].(string)
	if !ok || version != "draft-ver-123" {
		t.Fatalf("expected string version in delete body, got %#v", gotBody["version"])
	}
}

func TestDraftDeleteHumanOutput(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	restoreStdout := captureFile(t, &os.Stdout)

	cmd := DraftDeleteCmd{
		ID:           "msg_draft_123",
		DraftVersion: "draft-ver-123",
	}
	flags := &RootFlags{Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	stdout := restoreStdout()

	if !strings.Contains(stdout, "Draft deleted") {
		t.Fatalf("expected delete output, got %q", stdout)
	}
}

func TestDraftCreateRequiresAuthorBeforeClient(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		t.Fatal("client should not be created when author is missing")
		return nil, fmt.Errorf("unexpected client creation")
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftCreateCmd{
		Channel: "cha_v4x",
		Body:    "hello world",
		Inbox:   "inb_team",
	}
	flags := &RootFlags{Account: "test@example.com"}

	err := cmd.Run(flags)
	if err == nil {
		t.Fatal("expected missing author to fail")
	}

	if !strings.Contains(err.Error(), "--author is required") {
		t.Fatalf("expected missing author error, got %v", err)
	}
}

func TestDraftCreateRequiresInboxBeforeClient(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		t.Fatal("client should not be created when inbox is missing")
		return nil, fmt.Errorf("unexpected client creation")
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftCreateCmd{
		Channel: "cha_v4x",
		Body:    "hello world",
		Author:  "tea_author",
	}
	flags := &RootFlags{Account: "test@example.com"}

	err := cmd.Run(flags)
	if err == nil {
		t.Fatal("expected missing inbox to fail")
	}

	if !strings.Contains(err.Error(), "--inbox is required") {
		t.Fatalf("expected missing inbox error, got %v", err)
	}
}

func TestDraftCreateRoutesConversationWithExplicitAssignee(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	type request struct {
		method string
		path   string
		body   map[string]any
	}

	var requests []request

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := request{method: r.Method, path: r.URL.Path}
		if r.Body != nil {
			defer r.Body.Close()

			if r.ContentLength != 0 {
				if err := json.NewDecoder(r.Body).Decode(&req.body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
			}
		}
		requests = append(requests, req)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/channels/cha_v4x/drafts":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"type":"email",
				"is_inbound":false,
				"draft_mode":"shared",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"hello world",
				"text":"hello world",
				"subject":"Test draft",
				"_links":{"related":{"conversation":"https://api2.frontapp.com/conversations/cnv_999"}}
			}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/conversations/cnv_999":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	restoreStdout := captureFile(t, &os.Stdout)

	cmd := DraftCreateCmd{
		Channel:  "cha_v4x",
		To:       "alice@example.com",
		Subject:  "Test draft",
		Body:     "hello world",
		Author:   "tea_author",
		Inbox:    "inb_team",
		Assignee: "tea_assignee",
		Mode:     "shared",
	}
	flags := &RootFlags{Plain: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	stdout := restoreStdout()

	if len(requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requests))
	}

	postBody := requests[0].body
	if postBody["author_id"] != "tea_author" {
		t.Fatalf("expected author_id in create request, got %#v", postBody["author_id"])
	}

	if postBody["mode"] != "shared" {
		t.Fatalf("expected mode in create request, got %#v", postBody["mode"])
	}

	patchBody := requests[1].body
	if patchBody["inbox_id"] != "inb_team" {
		t.Fatalf("expected inbox_id in patch request, got %#v", patchBody["inbox_id"])
	}

	if patchBody["assignee_id"] != "tea_assignee" {
		t.Fatalf("expected assignee_id in patch request, got %#v", patchBody["assignee_id"])
	}

	if !strings.Contains(stdout, "Draft created: msg_draft_123") ||
		!strings.Contains(stdout, "conversation: cnv_999") ||
		!strings.Contains(stdout, "author: tea_author") ||
		!strings.Contains(stdout, "inbox: inb_team") ||
		!strings.Contains(stdout, "assignee: tea_assignee") ||
		!strings.Contains(stdout, "mode: shared") {
		t.Fatalf("expected success output with routing details, got %q", stdout)
	}
}

func TestDraftCreateDefaultsAssigneeToAuthorForReplyDrafts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	type request struct {
		method string
		path   string
		body   map[string]any
	}

	var requests []request

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := request{method: r.Method, path: r.URL.Path}
		if r.Body != nil {
			defer r.Body.Close()

			if r.ContentLength != 0 {
				if err := json.NewDecoder(r.Body).Decode(&req.body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
			}
		}
		requests = append(requests, req)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/conversations/cnv_123/drafts":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"type":"email",
				"is_inbound":false,
				"draft_mode":"private",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"reply body",
				"text":"reply body"
			}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/conversations/cnv_123":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftCreateCmd{
		ConvID:  "cnv_123",
		Channel: "cha_v4x",
		Body:    "reply body",
		Author:  "tea_author",
		Inbox:   "inb_team",
	}
	flags := &RootFlags{JSON: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requests))
	}

	postBody := requests[0].body
	if postBody["mode"] != "private" {
		t.Fatalf("expected default private mode, got %#v", postBody["mode"])
	}

	if postBody["channel_id"] != "cha_v4x" {
		t.Fatalf("expected channel_id in reply draft body, got %#v", postBody["channel_id"])
	}

	patchBody := requests[1].body
	if patchBody["assignee_id"] != "tea_author" {
		t.Fatalf("expected assignee_id to default to author, got %#v", patchBody["assignee_id"])
	}
}

func TestDraftCreateRequiresChannelForReplyDrafts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := (&DraftCreateCmd{
		ConvID: "cnv_123",
		Body:   "reply body",
		Author: "tea_author",
		Inbox:  "inb_team",
	}).Run(&RootFlags{JSON: true, Account: "test@example.com"})
	if err == nil || !strings.Contains(err.Error(), "--channel is required for reply drafts") {
		t.Fatalf("expected channel requirement error, got %v", err)
	}
}

func TestDraftCreateReturnsPartialSuccessErrorWhenConversationUpdateFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/channels/cha_v4x/drafts":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{
				"id":"msg_draft_123",
				"type":"email",
				"is_inbound":false,
				"draft_mode":"private",
				"version":"draft-ver-123",
				"created_at":1710000000,
				"body":"hello world",
				"text":"hello world",
				"_links":{"related":{"conversation":"https://api2.frontapp.com/conversations/cnv_999"}}
			}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/conversations/cnv_999":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"_error":{"message":"bad inbox"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := DraftCreateCmd{
		Channel: "cha_v4x",
		Body:    "hello world",
		Author:  "tea_author",
		Inbox:   "inb_team",
	}
	flags := &RootFlags{Account: "test@example.com"}

	err := cmd.Run(flags)
	if err == nil {
		t.Fatal("expected conversation update failure")
	}

	if !strings.Contains(err.Error(), "draft created: msg_draft_123") ||
		!strings.Contains(err.Error(), "conversation: cnv_999") {
		t.Fatalf("expected partial success details in error, got %v", err)
	}
}

func captureFile(t *testing.T, target **os.File) func() string {
	t.Helper()

	old := *target
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	*target = w

	var (
		once   sync.Once
		output string
	)

	restore := func() string {
		once.Do(func() {
			_ = w.Close()
			*target = old

			var buf bytes.Buffer
			if _, copyErr := io.Copy(&buf, r); copyErr != nil {
				t.Fatalf("copy output: %v", copyErr)
			}

			_ = r.Close()
			output = buf.String()
		})

		return output
	}

	t.Cleanup(func() {
		_ = restore()
	})

	return func() string {
		return restore()
	}
}
