package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dedene/frontapp-cli/internal/config"
	"github.com/dedene/frontapp-cli/internal/errfmt"
	"github.com/dedene/frontapp-cli/internal/markdown"
	"github.com/dedene/frontapp-cli/internal/output"
)

type MsgCmd struct {
	Get         MsgGetCmd         `cmd:"" help:"Get a message"`
	Send        MsgSendCmd        `cmd:"" help:"Send a new message"`
	Reply       MsgReplyCmd       `cmd:"" help:"Reply to a conversation"`
	Attachments MsgAttachmentsCmd `cmd:"" help:"List message attachments"`
	Attachment  MsgAttachmentCmd  `cmd:"" help:"Attachment operations"`
}

type MsgGetCmd struct {
	ID  string `arg:"" help:"Message ID"`
	Raw bool   `help:"Show raw body (no HTML conversion)"`
}

func (c *MsgGetCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(ctx, c.ID)
	if err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, msg)
	}

	direction := "Outbound"
	if msg.IsInbound {
		direction = "Inbound"
	}

	fmt.Fprintf(os.Stdout, "ID:        %s\n", msg.ID)
	fmt.Fprintf(os.Stdout, "Type:      %s\n", msg.Type)
	fmt.Fprintf(os.Stdout, "Direction: %s\n", direction)

	if msg.Subject != "" {
		fmt.Fprintf(os.Stdout, "Subject:   %s\n", msg.Subject)
	}

	if msg.Author != nil {
		author := msg.Author.Email
		if author == "" {
			author = msg.Author.Username
		}

		fmt.Fprintf(os.Stdout, "Author:    %s\n", author)
	}

	fmt.Fprintf(os.Stdout, "Date:      %s\n", output.FormatTimestamp(msg.CreatedAt))
	fmt.Fprintln(os.Stdout)

	switch {
	case c.Raw:
		fmt.Fprintln(os.Stdout, msg.Body)
	case msg.Text != "":
		fmt.Fprintln(os.Stdout, msg.Text)
	default:
		md, err := markdown.ToMarkdown(msg.Body)
		if err == nil && strings.TrimSpace(md) != "" {
			fmt.Fprintln(os.Stdout, md)
		} else {
			fmt.Fprintln(os.Stdout, msg.Body)
		}
	}

	return nil
}

type MsgSendCmd struct {
	Channel  string `required:"" help:"Channel ID to send from"`
	To       string `required:"" help:"Recipient address"`
	Subject  string `help:"Message subject"`
	Body     string `help:"Message body"`
	BodyFile string `help:"Read body from file" type:"existingfile"`
}

func (c *MsgSendCmd) Run(flags *RootFlags) error {
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

	if body == "" {
		return fmt.Errorf("body is required (use --body or --body-file)")
	}

	req := map[string]any{
		"to":   []string{c.To},
		"body": body,
	}

	if c.Subject != "" {
		req["subject"] = c.Subject
	}

	var result map[string]any
	if err := client.Post(ctx, fmt.Sprintf("/channels/%s/messages", c.Channel), req, &result); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintln(os.Stdout, "Message sent successfully")

	return nil
}

type MsgReplyCmd struct {
	ConvID    string `arg:"" help:"Conversation ID to reply to"`
	Body      string `help:"Reply body"`
	BodyFile  string `help:"Read body from file" type:"existingfile"`
	InReplyTo string `help:"Message ID to reply to (for threading)"`
}

func (c *MsgReplyCmd) Run(flags *RootFlags) error {
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

	if body == "" {
		return fmt.Errorf("body is required (use --body or --body-file)")
	}

	req := map[string]any{
		"body": body,
		"type": "reply",
	}

	if c.InReplyTo != "" {
		req["in_reply_to_message_id"] = c.InReplyTo
	}

	var result map[string]any
	if err := client.Post(ctx, fmt.Sprintf("/conversations/%s/messages", c.ConvID), req, &result); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintln(os.Stdout, "Reply sent successfully")

	return nil
}

type MsgAttachmentsCmd struct {
	ID string `arg:"" help:"Message ID"`
}

func (c *MsgAttachmentsCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(ctx, c.ID)
	if err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, map[string]any{"attachments": msg.Attachments})
	}

	if len(msg.Attachments) == 0 {
		fmt.Fprintln(os.Stdout, "No attachments")

		return nil
	}

	tbl := output.NewTableWriter(os.Stdout, mode.Plain)
	tbl.AddRow("ID", "FILENAME", "TYPE", "SIZE")

	for _, att := range msg.Attachments {
		tbl.AddRow(att.ID, att.Filename, att.ContentType, fmt.Sprintf("%d", att.Size))
	}

	return tbl.Flush()
}

type MsgAttachmentCmd struct {
	Download MsgAttachmentDownloadCmd `cmd:"" help:"Download an attachment"`
}

type MsgAttachmentDownloadCmd struct {
	ID     string `arg:"" help:"Attachment ID"`
	Output string `short:"o" help:"Output file path"`
}

func (c *MsgAttachmentDownloadCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if strings.TrimSpace(c.Output) == "" {
		return fmt.Errorf("--output is required")
	}

	path, err := config.ExpandPath(c.Output)
	if err != nil {
		return err
	}

	f, err := os.Create(path) //nolint:gosec // Path is cleaned by config.ExpandPath
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	if err := client.Download(ctx, fmt.Sprintf("/download/%s", c.ID), f); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Attachment saved to %s\n", path)

	return nil
}
