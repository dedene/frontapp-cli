package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/dedene/frontapp-cli/internal/api"
	"github.com/dedene/frontapp-cli/internal/errfmt"
	"github.com/dedene/frontapp-cli/internal/output"
)

const (
	draftModePrivate = "private"
	draftModeShared  = "shared"
)

type DraftCmd struct {
	Create DraftCreateCmd `cmd:"" help:"Create a draft"`
	List   DraftListCmd   `cmd:"" help:"List drafts in a conversation"`
	Get    DraftGetCmd    `cmd:"" help:"Get a draft"`
	Update DraftUpdateCmd `cmd:"" help:"Update a draft"`
	Delete DraftDeleteCmd `cmd:"" help:"Delete a draft"`
}

type DraftCreateCmd struct {
	ConvID           string `arg:"" help:"Conversation ID (for reply drafts)" optional:""`
	Channel          string `help:"Channel ID"`
	To               string `help:"Recipient (for new message drafts)"`
	Author           string `help:"Teammate ID to create the draft as"`
	Inbox            string `help:"Inbox ID to move the draft conversation to"`
	Assignee         string `help:"Teammate ID to assign the draft conversation to (defaults to --author)"`
	Mode             string `help:"Draft mode: private or shared" default:"private"`
	DefaultSignature bool   `help:"Add the teammate's default signature" default:"true" negatable:""`
	Signature        string `help:"Explicit signature ID to attach"`
	Subject          string `help:"Draft subject"`
	Body             string `help:"Draft body"`
	BodyFile         string `help:"Read body from file" type:"existingfile"`
}

func (c *DraftCreateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	if strings.TrimSpace(c.Author) == "" {
		return fmt.Errorf("--author is required")
	}

	if strings.TrimSpace(c.Inbox) == "" {
		return fmt.Errorf("--inbox is required")
	}

	draftMode := strings.TrimSpace(c.Mode)
	if draftMode == "" {
		draftMode = draftModePrivate
	}

	if draftMode != draftModePrivate && draftMode != draftModeShared {
		return fmt.Errorf("--mode must be private or shared")
	}

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	body := c.Body
	if c.BodyFile != "" {
		var data []byte
		data, err = os.ReadFile(c.BodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}

		body = string(data)
	}

	req := map[string]any{
		"author_id": c.Author,
		"body":      body,
		"mode":      draftMode,
	}

	if signatureID := strings.TrimSpace(c.Signature); signatureID != "" {
		req["signature_id"] = signatureID
	} else if c.DefaultSignature {
		req["should_add_default_signature"] = true
	}

	if c.Subject != "" {
		req["subject"] = c.Subject
	}

	if c.To != "" {
		req["to"] = []string{c.To}
	}

	var createPath string
	switch {
	case c.ConvID != "":
		if strings.TrimSpace(c.Channel) == "" {
			return fmt.Errorf("--channel is required for reply drafts")
		}

		req["channel_id"] = c.Channel
		createPath = fmt.Sprintf("/conversations/%s/drafts", c.ConvID)
	case c.Channel != "":
		createPath = fmt.Sprintf("/channels/%s/drafts", c.Channel)
	default:
		return fmt.Errorf("either conversation ID or --channel is required")
	}

	var result api.DraftMessage
	err = client.Post(ctx, createPath, req, &result)
	if err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	conversationID, err := conversationIDFromDraft(result, c.ConvID)
	if err != nil {
		return fmt.Errorf("draft created: %s; conversation lookup failed: %w", result.ID, err)
	}

	assigneeID := strings.TrimSpace(c.Assignee)
	if assigneeID == "" {
		assigneeID = c.Author
	}

	patchReq := map[string]any{
		"assignee_id": assigneeID,
		"inbox_id":    c.Inbox,
	}

	if err := client.Patch(ctx, "/conversations/"+conversationID, patchReq, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return fmt.Errorf("draft created: %s; conversation: %s; update failed: %w", result.ID, conversationID, err)
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stdout, "Draft created: %s\n", result.ID)
	fmt.Fprintf(os.Stdout, "conversation: %s\n", conversationID)
	fmt.Fprintf(os.Stdout, "author: %s\n", c.Author)
	fmt.Fprintf(os.Stdout, "inbox: %s\n", c.Inbox)
	fmt.Fprintf(os.Stdout, "assignee: %s\n", assigneeID)
	fmt.Fprintf(os.Stdout, "mode: %s\n", draftMode)

	return nil
}

type DraftListCmd struct {
	ConvID string `arg:"" help:"Conversation ID"`
}

func (c *DraftListCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	var resp api.ListResponse[api.DraftMessage]
	if err := client.Get(ctx, fmt.Sprintf("/conversations/%s/drafts", c.ConvID), &resp); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, resp)
	}

	if len(resp.Results) == 0 {
		fmt.Fprintln(os.Stdout, "No drafts found.")

		return nil
	}

	tbl := output.NewTableWriter(os.Stdout, mode.Plain)
	tbl.AddRow("ID", "VERSION", "SUBJECT", "CREATED")

	for _, draft := range resp.Results {
		tbl.AddRow(
			draft.ID,
			draft.Version,
			draft.Subject,
			output.FormatTimestamp(draft.CreatedAt),
		)
	}

	return tbl.Flush()
}

type DraftGetCmd struct {
	ID string `arg:"" help:"Draft ID"`
}

func (c *DraftGetCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	var draft api.DraftMessage
	if err := client.Get(ctx, "/messages/"+c.ID, &draft); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if draft.DraftMode == "" {
		return fmt.Errorf("message %s is not a draft", c.ID)
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, draft)
	}

	fmt.Fprintf(os.Stdout, "ID:      %s\n", draft.ID)
	fmt.Fprintf(os.Stdout, "Version: %s\n", draft.Version)

	if draft.Subject != "" {
		fmt.Fprintf(os.Stdout, "Subject: %s\n", draft.Subject)
	}

	fmt.Fprintf(os.Stdout, "Created: %s\n", output.FormatTimestamp(draft.CreatedAt))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, draft.Body)

	return nil
}

type DraftUpdateCmd struct {
	ID           string `arg:"" help:"Draft ID"`
	Channel      string `help:"Channel ID required by Front for some draft updates"`
	Body         string `help:"New body"`
	BodyFile     string `help:"Read body from file" type:"existingfile"`
	Subject      string `help:"New subject"`
	DraftVersion string `required:"" name:"draft-version" help:"Current version token (for optimistic locking)"`
}

func (c *DraftUpdateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	body := c.Body
	if c.BodyFile != "" {
		data, err := os.ReadFile(c.BodyFile)
		if err != nil {
			return fmt.Errorf("read body file: %w", err)
		}

		body = string(data)
	}

	req := map[string]any{
		"version": c.DraftVersion,
	}

	if body != "" {
		req["body"] = body
	}

	if c.Channel != "" {
		req["channel_id"] = c.Channel
	}

	if c.Subject != "" {
		req["subject"] = c.Subject
	}

	var result api.DraftMessage
	if err := client.Patch(ctx, "/drafts/"+c.ID, req, &result); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stdout, "Draft updated (new version: %s)\n", result.Version)

	return nil
}

type DraftDeleteCmd struct {
	ID           string `arg:"" help:"Draft ID"`
	DraftVersion string `required:"" name:"draft-version" help:"Current version token (required by Front to delete the draft)"`
}

func (c *DraftDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	req := map[string]any{
		"version": c.DraftVersion,
	}

	if err := client.DeleteWithBody(ctx, "/drafts/"+c.ID, req); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintln(os.Stdout, "Draft deleted")

	return nil
}

func conversationIDFromDraft(draft api.DraftMessage, fallback string) (string, error) {
	conversationURL := strings.TrimSpace(draft.Links.Related["conversation"])
	if conversationURL == "" {
		if strings.TrimSpace(fallback) != "" {
			return strings.TrimSpace(fallback), nil
		}

		return "", fmt.Errorf("draft response missing conversation link")
	}

	parsed, err := url.Parse(conversationURL)
	if err != nil {
		return "", fmt.Errorf("parse conversation link: %w", err)
	}

	conversationID := path.Base(strings.TrimRight(parsed.Path, "/"))
	if conversationID == "" || conversationID == "." || conversationID == "/" {
		return "", fmt.Errorf("conversation link missing id")
	}

	if err := api.ValidateIDPrefix(conversationID, "cnv_"); err != nil {
		return "", fmt.Errorf("conversation link returned unexpected id: %w", err)
	}

	return conversationID, nil
}
