package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/dedene/frontapp-cli/internal/api"
)

func TestConvSearchEncodesQuery(t *testing.T) {
	var gotPath string
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("q")
		_, _ = io.WriteString(w, `{"_results":[]}`)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := ConvSearchCmd{Query: "from:me project update", Limit: 10}
	flags := &RootFlags{JSON: true, Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if gotPath != "/conversations/search" {
		t.Fatalf("expected path /conversations/search, got %s", gotPath)
	}

	if gotQuery != "from:me project update" {
		t.Fatalf("unexpected query: %q", gotQuery)
	}
}

func TestConvTagSendsTagIDs(t *testing.T) {
	var gotBody map[string][]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/conversations/cnv_123/tags" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	cmd := ConvTagCmd{ID: "cnv_123", TagID: "tag_abc"}
	flags := &RootFlags{Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if gotBody == nil || len(gotBody["tag_ids"]) != 1 || gotBody["tag_ids"][0] != "tag_abc" {
		t.Fatalf("unexpected body: %#v", gotBody)
	}
}

func TestConvArchiveIDsFromStdin(t *testing.T) {
	var seen []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}

		seen = append(seen, strings.TrimPrefix(r.URL.Path, "/conversations/"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	old := newClientFromAuth
	newClientFromAuth = func(_, _ string) (*api.Client, error) {
		return api.NewClientWithBaseURL(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), srv.URL), nil
	}
	t.Cleanup(func() { newClientFromAuth = old })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	_, _ = w.WriteString("cnv_1\ncnv_2\n")
	_ = w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	cmd := ConvArchiveCmd{IDsFrom: "-"}
	flags := &RootFlags{Account: "test@example.com"}

	if err := cmd.Run(flags); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(seen) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(seen))
	}
}
