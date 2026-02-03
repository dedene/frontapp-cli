package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/dedene/frontapp-cli/internal/auth"
)

const (
	BaseURL     = "https://api2.frontapp.com"
	UserAgent   = "frontcli/0.1.0"
	ContentType = "application/json"
)

var errWriterRequired = errors.New("writer is required")

// Client is the Front API client.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
	rateLimiter *RateLimiter
}

// NewClient creates a new API client with the given token source.
func NewClient(ts oauth2.TokenSource) *Client {
	return &Client{
		baseURL:     BaseURL,
		tokenSource: ts,
		httpClient: &http.Client{
			Transport: NewRetryTransport(http.DefaultTransport),
		},
		rateLimiter: NewRateLimiter(),
	}
}

// NewClientWithBaseURL creates a new API client with a custom base URL.
func NewClientWithBaseURL(ts oauth2.TokenSource, baseURL string) *Client {
	client := NewClient(ts)
	if strings.TrimSpace(baseURL) != "" {
		client.baseURL = strings.TrimRight(baseURL, "/")
	}

	return client
}

// NewClientFromAuth creates a client using stored auth credentials.
func NewClientFromAuth(clientName, email string) (*Client, error) {
	store, err := auth.OpenDefault()
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	ts := auth.NewTokenSource(clientName, email, store)

	return NewClient(ts), nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, out interface{}) error {
	reqURL := c.baseURL + path

	for attempt := 0; attempt < 2; attempt++ {
		if c.rateLimiter != nil {
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return err
			}
		}

		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		// Get token and set auth header
		tok, err := c.tokenSource.Token()
		if err != nil {
			return &AuthError{Err: err}
		}

		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", ContentType)

		if body != nil {
			req.Header.Set("Content-Type", ContentType)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("do request: %w", err)
		}

		if c.rateLimiter != nil {
			c.rateLimiter.UpdateFromHeaders(resp.Header)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			if ts, ok := c.tokenSource.(*auth.TokenSource); ok {
				ts.Invalidate()
			}

			drainAndClose(resp.Body)

			if attempt == 0 {
				continue
			}

			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    "unauthorized",
				Details:    "token may be expired; try logging in again",
			}
		}

		if resp.StatusCode == http.StatusNotFound {
			defer resp.Body.Close()

			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    "not found",
			}
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := 0
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				retryAfter, _ = strconv.Atoi(ra)
			}

			defer resp.Body.Close()

			return &RateLimitError{RetryAfter: retryAfter}
		}

		if resp.StatusCode >= 400 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    http.StatusText(resp.StatusCode),
				Details:    string(bodyBytes),
			}
		}

		if out != nil && resp.StatusCode != http.StatusNoContent {
			defer resp.Body.Close()

			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			return nil
		}

		_ = resp.Body.Close()

		return nil
	}

	return &APIError{
		StatusCode: http.StatusUnauthorized,
		Message:    "unauthorized",
		Details:    "token may be expired; try logging in again",
	}
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, out interface{}) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}, out interface{}) error {
	var bodyBytes []byte

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}

		bodyBytes = data
	}

	return c.do(ctx, http.MethodPost, path, bodyBytes, out)
}

// Patch performs a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body interface{}, out interface{}) error {
	var bodyBytes []byte

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}

		bodyBytes = data
	}

	return c.do(ctx, http.MethodPatch, path, bodyBytes, out)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// Download performs a GET request and writes the response body to the writer.
func (c *Client) Download(ctx context.Context, path string, w io.Writer) error {
	if w == nil {
		return errWriterRequired
	}

	reqURL := c.baseURL + path

	for attempt := 0; attempt < 2; attempt++ {
		if c.rateLimiter != nil {
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return err
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		tok, err := c.tokenSource.Token()
		if err != nil {
			return &AuthError{Err: err}
		}

		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		req.Header.Set("User-Agent", UserAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("do request: %w", err)
		}

		if c.rateLimiter != nil {
			c.rateLimiter.UpdateFromHeaders(resp.Header)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			if ts, ok := c.tokenSource.(*auth.TokenSource); ok {
				ts.Invalidate()
			}

			if attempt == 0 {
				drainAndClose(resp.Body)
				continue
			}

			_ = resp.Body.Close()

			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    "unauthorized",
				Details:    "token may be expired; try logging in again",
			}
		}

		if resp.StatusCode >= 400 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    http.StatusText(resp.StatusCode),
				Details:    string(bodyBytes),
			}
		}

		if _, err := io.Copy(w, resp.Body); err != nil {
			_ = resp.Body.Close()
			return fmt.Errorf("download body: %w", err)
		}

		_ = resp.Body.Close()

		return nil
	}

	return &APIError{
		StatusCode: http.StatusUnauthorized,
		Message:    "unauthorized",
		Details:    "token may be expired; try logging in again",
	}
}

// Me returns the authenticated user's info.
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var me Me
	if err := c.Get(ctx, "/me", &me); err != nil {
		return nil, err
	}

	return &me, nil
}

// ListConversations lists conversations with optional filters.
func (c *Client) ListConversations(ctx context.Context, opts ListConversationsOptions) (*ListResponse[Conversation], error) {
	path := "/conversations?" + opts.Query()

	var resp ListResponse[Conversation]
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetConversation gets a single conversation by ID.
func (c *Client) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	var conv Conversation
	if err := c.Get(ctx, "/conversations/"+id, &conv); err != nil {
		return nil, err
	}

	return &conv, nil
}

// ListConversationMessages lists messages in a conversation.
func (c *Client) ListConversationMessages(ctx context.Context, convID string, limit int) (*ListResponse[Message], error) {
	path := fmt.Sprintf("/conversations/%s/messages", convID)
	if limit > 0 {
		path += fmt.Sprintf("?limit=%d", limit)
	}

	var resp ListResponse[Message]
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetMessage gets a single message by ID.
func (c *Client) GetMessage(ctx context.Context, id string) (*Message, error) {
	var msg Message
	if err := c.Get(ctx, "/messages/"+id, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// ListInboxes lists all inboxes.
func (c *Client) ListInboxes(ctx context.Context) (*ListResponse[Inbox], error) {
	var resp ListResponse[Inbox]
	if err := c.Get(ctx, "/inboxes", &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetInbox gets a single inbox by ID.
func (c *Client) GetInbox(ctx context.Context, id string) (*Inbox, error) {
	var inbox Inbox
	if err := c.Get(ctx, "/inboxes/"+id, &inbox); err != nil {
		return nil, err
	}

	return &inbox, nil
}

// ListTags lists all tags.
func (c *Client) ListTags(ctx context.Context) (*ListResponse[Tag], error) {
	var resp ListResponse[Tag]
	if err := c.Get(ctx, "/tags", &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetTag gets a single tag by ID.
func (c *Client) GetTag(ctx context.Context, id string) (*Tag, error) {
	var tag Tag
	if err := c.Get(ctx, "/tags/"+id, &tag); err != nil {
		return nil, err
	}

	return &tag, nil
}

// ListTeammates lists all teammates.
func (c *Client) ListTeammates(ctx context.Context) (*ListResponse[Teammate], error) {
	var resp ListResponse[Teammate]
	if err := c.Get(ctx, "/teammates", &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetTeammate gets a single teammate by ID.
func (c *Client) GetTeammate(ctx context.Context, id string) (*Teammate, error) {
	var tm Teammate
	if err := c.Get(ctx, "/teammates/"+id, &tm); err != nil {
		return nil, err
	}

	return &tm, nil
}

// ListChannels lists all channels.
func (c *Client) ListChannels(ctx context.Context) (*ListResponse[Channel], error) {
	var resp ListResponse[Channel]
	if err := c.Get(ctx, "/channels", &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetChannel gets a single channel by ID.
func (c *Client) GetChannel(ctx context.Context, id string) (*Channel, error) {
	var ch Channel
	if err := c.Get(ctx, "/channels/"+id, &ch); err != nil {
		return nil, err
	}

	return &ch, nil
}

// ListContacts lists contacts.
func (c *Client) ListContacts(ctx context.Context, limit int) (*ListResponse[Contact], error) {
	path := "/contacts"
	if limit > 0 {
		path += fmt.Sprintf("?limit=%d", limit)
	}

	var resp ListResponse[Contact]
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListContactsPage fetches a page of contacts using a page token.
func (c *Client) ListContactsPage(ctx context.Context, pageURL string) (*ListResponse[Contact], error) {
	// pageURL is a full URL; extract path+query
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("parse page URL: %w", err)
	}

	path := parsed.Path
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}

	var resp ListResponse[Contact]
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetContact gets a single contact by ID.
func (c *Client) GetContact(ctx context.Context, id string) (*Contact, error) {
	var contact Contact
	if err := c.Get(ctx, "/contacts/"+id, &contact); err != nil {
		return nil, err
	}

	return &contact, nil
}

// ListConversationsOptions contains options for listing conversations.
type ListConversationsOptions struct {
	InboxID   string
	TagID     string
	Statuses  []string // assigned, unassigned, archived, trashed, snoozed
	Limit     int
	PageToken string
	SortOrder string // asc, desc (default: desc = most recent first)
}

// ParseStatus converts a user-friendly status to API statuses.
// "open" expands to ["assigned", "unassigned"].
func ParseStatus(status string) []string {
	if status == "" {
		return nil
	}

	if status == "open" {
		return []string{"assigned", "unassigned"}
	}

	return []string{status}
}

func (o ListConversationsOptions) Query() string {
	params := url.Values{}

	if o.InboxID != "" {
		params.Set("q[inbox_id]", o.InboxID)
	}

	if o.TagID != "" {
		params.Set("q[tag_id]", o.TagID)
	}

	for _, status := range o.Statuses {
		params.Add("q[statuses][]", status)
	}

	if o.Limit > 0 {
		params.Set("limit", strconv.Itoa(o.Limit))
	}

	if o.PageToken != "" {
		params.Set("page_token", o.PageToken)
	}

	if o.SortOrder != "" && o.SortOrder != "-" {
		params.Set("sort_by", "date")
		params.Set("sort_order", o.SortOrder)
	}

	return params.Encode()
}
