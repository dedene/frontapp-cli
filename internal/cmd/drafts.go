package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dedene/frontapp-cli/internal/api"
	"github.com/dedene/frontapp-cli/internal/errfmt"
	"github.com/dedene/frontapp-cli/internal/output"
)

type DraftCmd struct {
	Create DraftCreateCmd `cmd:"" help:"Create a draft"`
	List   DraftListCmd   `cmd:"" help:"List drafts in a conversation"`
	Get    DraftGetCmd    `cmd:"" help:"Get a draft"`
	Update DraftUpdateCmd `cmd:"" help:"Update a draft"`
	Delete DraftDeleteCmd `cmd:"" help:"Delete a draft"`
}

type DraftCreateCmd struct {
	ConvID   string `arg:"" help:"Conversation ID (for reply drafts)" optional:""`
	Channel  string `help:"Channel ID (for new message drafts)"`
	To       string `help:"Recipient (for new message drafts)"`
	Subject  string `help:"Draft subject"`
	Body     string `help:"Draft body"`
	BodyFile string `help:"Read body from file" type:"existingfile"`
}

func (c *DraftCreateCmd) Run(flags *RootFlags) error {
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
		"body": body,
	}

	if c.Subject != "" {
		req["subject"] = c.Subject
	}

	if c.To != "" {
		req["to"] = []string{c.To}
	}

	var path string
	switch {
	case c.ConvID != "":
		path = fmt.Sprintf("/conversations/%s/drafts", c.ConvID)
	case c.Channel != "":
		path = fmt.Sprintf("/channels/%s/drafts", c.Channel)
	default:
		return fmt.Errorf("either conversation ID or --channel is required")
	}

	var result api.Draft
	if err := client.Post(ctx, path, req, &result); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stdout, "Draft created: %s\n", result.ID)

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

	var resp api.ListResponse[api.Draft]
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
			fmt.Sprintf("%d", draft.Version),
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

	var draft api.Draft
	if err := client.Get(ctx, "/drafts/"+c.ID, &draft); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, draft)
	}

	fmt.Fprintf(os.Stdout, "ID:      %s\n", draft.ID)
	fmt.Fprintf(os.Stdout, "Version: %d\n", draft.Version)

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
	Body         string `help:"New body"`
	BodyFile     string `help:"Read body from file" type:"existingfile"`
	Subject      string `help:"New subject"`
	DraftVersion int    `required:"" name:"draft-version" help:"Current version number (for optimistic locking)"`
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

	if c.Subject != "" {
		req["subject"] = c.Subject
	}

	var result api.Draft
	if err := client.Patch(ctx, "/drafts/"+c.ID, req, &result); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stdout, "Draft updated (new version: %d)\n", result.Version)

	return nil
}

type DraftDeleteCmd struct {
	ID string `arg:"" help:"Draft ID"`
}

func (c *DraftDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if err := client.Delete(ctx, "/drafts/"+c.ID); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintln(os.Stdout, "Draft deleted")

	return nil
}
