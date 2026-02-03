package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dedene/frontapp-cli/internal/api"
	"github.com/dedene/frontapp-cli/internal/errfmt"
	"github.com/dedene/frontapp-cli/internal/output"
)

type ConvArchiveCmd struct {
	IDs     []string `arg:"" help:"Conversation IDs to archive"`
	IDsFrom string   `help:"Read conversation IDs from stdin (use '-' for stdin)"`
}

func (c *ConvArchiveCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	ids, err := collectIDs(c.IDs, c.IDsFrom)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return fmt.Errorf("no conversation IDs provided")
	}

	for _, id := range ids {
		if err := client.Patch(ctx, "/conversations/"+id, map[string]string{"status": "archived"}, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to archive %s: %v\n", id, err)
		} else {
			fmt.Fprintf(os.Stdout, "Archived %s\n", id)
		}
	}

	return nil
}

type ConvOpenCmd struct {
	IDs     []string `arg:"" help:"Conversation IDs to open"`
	IDsFrom string   `help:"Read conversation IDs from stdin (use '-' for stdin)"`
}

func (c *ConvOpenCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	ids, err := collectIDs(c.IDs, c.IDsFrom)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return fmt.Errorf("no conversation IDs provided")
	}

	for _, id := range ids {
		if err := client.Patch(ctx, "/conversations/"+id, map[string]string{"status": "open"}, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", id, err)
		} else {
			fmt.Fprintf(os.Stdout, "Opened %s\n", id)
		}
	}

	return nil
}

type ConvTrashCmd struct {
	IDs     []string `arg:"" help:"Conversation IDs to trash"`
	IDsFrom string   `help:"Read conversation IDs from stdin (use '-' for stdin)"`
}

func (c *ConvTrashCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	ids, err := collectIDs(c.IDs, c.IDsFrom)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return fmt.Errorf("no conversation IDs provided")
	}

	for _, id := range ids {
		if err := client.Patch(ctx, "/conversations/"+id, map[string]string{"status": "trashed"}, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to trash %s: %v\n", id, err)
		} else {
			fmt.Fprintf(os.Stdout, "Trashed %s\n", id)
		}
	}

	return nil
}

type ConvAssignCmd struct {
	ID string `arg:"" help:"Conversation ID"`
	To string `required:"" help:"Teammate ID to assign to"`
}

func (c *ConvAssignCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if err := client.Patch(ctx, "/conversations/"+c.ID, map[string]string{"assignee_id": c.To}, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Assigned %s to %s\n", c.ID, c.To)

	return nil
}

type ConvUnassignCmd struct {
	ID string `arg:"" help:"Conversation ID"`
}

func (c *ConvUnassignCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if err := client.Patch(ctx, "/conversations/"+c.ID, map[string]any{"assignee_id": nil}, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Unassigned %s\n", c.ID)

	return nil
}

type ConvSnoozeCmd struct {
	ID       string `arg:"" help:"Conversation ID"`
	Until    string `help:"Snooze until (RFC3339 timestamp)"`
	Duration string `help:"Snooze duration (e.g. 2h, 30m)"`
}

func (c *ConvSnoozeCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	until := strings.TrimSpace(c.Until)
	if strings.TrimSpace(c.Duration) != "" {
		if until != "" {
			return fmt.Errorf("use either --until or --duration, not both")
		}

		d, err := time.ParseDuration(strings.TrimSpace(c.Duration))
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}

		until = time.Now().Add(d).UTC().Format(time.RFC3339)
	}

	if until == "" {
		return fmt.Errorf("either --until or --duration is required")
	}

	req := map[string]string{"scheduled_at": until}

	if err := client.Patch(ctx, fmt.Sprintf("/conversations/%s/reminders", c.ID), req, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Snoozed %s until %s\n", c.ID, until)

	return nil
}

type ConvUnsnoozeCmd struct {
	ID string `arg:"" help:"Conversation ID"`
}

func (c *ConvUnsnoozeCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	req := map[string]any{"scheduled_at": nil}

	if err := client.Patch(ctx, fmt.Sprintf("/conversations/%s/reminders", c.ID), req, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Unsnoozed %s\n", c.ID)

	return nil
}

type ConvFollowersCmd struct {
	ID string `arg:"" help:"Conversation ID"`
}

func (c *ConvFollowersCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	var resp api.ListResponse[api.Teammate]
	if err := client.Get(ctx, fmt.Sprintf("/conversations/%s/followers", c.ID), &resp); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, resp)
	}

	if len(resp.Results) == 0 {
		fmt.Fprintln(os.Stdout, "No followers found.")

		return nil
	}

	tbl := output.NewTableWriter(os.Stdout, mode.Plain)
	tbl.AddRow("ID", "EMAIL", "NAME")

	for _, tm := range resp.Results {
		tbl.AddRow(output.FormatTeammate(tm)...)
	}

	return tbl.Flush()
}

type ConvFollowCmd struct {
	ID   string `arg:"" help:"Conversation ID"`
	User string `help:"Teammate ID to follow as"`
}

func (c *ConvFollowCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	var body map[string]string
	if strings.TrimSpace(c.User) != "" {
		body = map[string]string{"teammate_id": c.User}
	}

	if err := client.Post(ctx, fmt.Sprintf("/conversations/%s/followers", c.ID), body, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if c.User != "" {
		fmt.Fprintf(os.Stdout, "Added follower %s to %s\n", c.User, c.ID)
	} else {
		fmt.Fprintf(os.Stdout, "Followed %s\n", c.ID)
	}

	return nil
}

type ConvUnfollowCmd struct {
	ID   string `arg:"" help:"Conversation ID"`
	User string `help:"Teammate ID to unfollow"`
}

func (c *ConvUnfollowCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/conversations/%s/followers", c.ID)
	if strings.TrimSpace(c.User) != "" {
		path = fmt.Sprintf("/conversations/%s/followers/%s", c.ID, c.User)
	}

	if err := client.Delete(ctx, path); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if c.User != "" {
		fmt.Fprintf(os.Stdout, "Removed follower %s from %s\n", c.User, c.ID)
	} else {
		fmt.Fprintf(os.Stdout, "Unfollowed %s\n", c.ID)
	}

	return nil
}

type ConvTagCmd struct {
	ID    string `arg:"" help:"Conversation ID"`
	TagID string `arg:"" help:"Tag ID to add"`
}

func (c *ConvTagCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	payload := map[string][]string{"tag_ids": {c.TagID}}
	if err := client.Post(ctx, fmt.Sprintf("/conversations/%s/tags", c.ID), payload, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Tagged %s with %s\n", c.ID, c.TagID)

	return nil
}

type ConvUntagCmd struct {
	ID    string `arg:"" help:"Conversation ID"`
	TagID string `arg:"" help:"Tag ID to remove"`
}

func (c *ConvUntagCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if err := client.Delete(ctx, fmt.Sprintf("/conversations/%s/tags/%s", c.ID, c.TagID)); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Untagged %s from %s\n", c.TagID, c.ID)

	return nil
}

type ConvUpdateCmd struct {
	ID     string   `arg:"" help:"Conversation ID"`
	Fields []string `help:"Custom field update (key=value)" name:"field"`
}

func (c *ConvUpdateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if len(c.Fields) == 0 {
		return fmt.Errorf("at least one --field key=value is required")
	}

	customFields := map[string]string{}
	for _, raw := range c.Fields {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid field format: %s", raw)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return fmt.Errorf("field name cannot be empty")
		}

		customFields[key] = value
	}

	req := map[string]any{
		"custom_fields": customFields,
	}

	if err := client.Patch(ctx, "/conversations/"+c.ID, req, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Updated %s\n", c.ID)

	return nil
}

func collectIDs(ids []string, idsFrom string) ([]string, error) {
	fromIDs, err := readIDsFromInput(idsFrom)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(ids)+len(fromIDs))
	out = append(out, ids...)
	out = append(out, fromIDs...)

	return out, nil
}
