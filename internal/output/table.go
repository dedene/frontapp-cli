package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/dedene/frontapp-cli/internal/api"
)

// Table provides simple table output using tabwriter.
type Table struct {
	w *tabwriter.Writer
}

type TableWriter interface {
	AddRow(cols ...string)
	Flush() error
}

type PlainTable struct {
	w io.Writer
}

func (t *PlainTable) AddRow(cols ...string) {
	fmt.Fprintln(t.w, strings.Join(cols, "\t"))
}

func (t *PlainTable) Flush() error {
	return nil
}

func NewTable(out io.Writer) *Table {
	return &Table{
		w: tabwriter.NewWriter(out, 0, 0, 2, ' ', 0),
	}
}

func NewPlainTable(out io.Writer) *PlainTable {
	return &PlainTable{w: out}
}

func NewTableWriter(out io.Writer, plain bool) TableWriter {
	if plain {
		return NewPlainTable(out)
	}

	return NewTable(out)
}

func (t *Table) AddRow(cols ...string) {
	fmt.Fprintln(t.w, strings.Join(cols, "\t"))
}

func (t *Table) Flush() error {
	if err := t.w.Flush(); err != nil {
		return fmt.Errorf("flush table: %w", err)
	}

	return nil
}

// FormatConversation formats a conversation for table output.
func FormatConversation(conv api.Conversation) []string {
	assignee := "-"
	if conv.Assignee != nil {
		assignee = conv.Assignee.Email
		if assignee == "" {
			assignee = conv.Assignee.Username
		}
	}

	subject := conv.Subject
	if len(subject) > 50 {
		subject = subject[:47] + "..."
	}

	return []string{
		conv.ID,
		conv.Status,
		assignee,
		subject,
		FormatTimestamp(conv.CreatedAt),
	}
}

// FormatConversationWithUpdated formats a conversation with UPDATED column.
// Shows waiting_since for snoozed, otherwise shows created_at.
func FormatConversationWithUpdated(conv api.Conversation) []string {
	assignee := "-"
	if conv.Assignee != nil {
		assignee = conv.Assignee.Email
		if assignee == "" {
			assignee = conv.Assignee.Username
		}
	}

	subject := conv.Subject
	if len(subject) > 50 {
		subject = subject[:47] + "..."
	}

	// Use waiting_since if available (snooze/activity time), else created_at
	updated := conv.WaitingSince
	if updated == 0 {
		updated = conv.CreatedAt
	}

	return []string{
		conv.ID,
		conv.Status,
		assignee,
		subject,
		FormatTimestamp(conv.CreatedAt),
		FormatTimestamp(updated),
	}
}

// FormatMessage formats a message for table output.
func FormatMessage(msg api.Message) []string {
	direction := "OUT"
	if msg.IsInbound {
		direction = "IN"
	}

	author := "-"
	if msg.Author != nil {
		author = msg.Author.Email
		if author == "" {
			author = msg.Author.Username
		}
	}

	blurb := msg.Blurb
	if len(blurb) > 60 {
		blurb = blurb[:57] + "..."
	}

	return []string{
		msg.ID,
		direction,
		author,
		blurb,
		FormatTimestamp(msg.CreatedAt),
	}
}

// FormatTag formats a tag for table output.
func FormatTag(tag api.Tag) []string {
	return []string{
		tag.ID,
		tag.Name,
		tag.Highlight,
	}
}

// FormatInbox formats an inbox for table output.
func FormatInbox(inbox api.Inbox) []string {
	return []string{
		inbox.ID,
		inbox.Name,
	}
}

// FormatTeammate formats a teammate for table output.
func FormatTeammate(tm api.Teammate) []string {
	name := strings.TrimSpace(tm.FirstName + " " + tm.LastName)
	if name == "" {
		name = tm.Username
	}

	return []string{
		tm.ID,
		tm.Email,
		name,
	}
}

// FormatContact formats a contact for table output.
func FormatContact(contact api.Contact) []string {
	handle := "-"
	if len(contact.Handles) > 0 {
		handle = contact.Handles[0].Handle
	}

	return []string{
		contact.ID,
		contact.Name,
		handle,
	}
}

// FormatChannel formats a channel for table output.
func FormatChannel(ch api.Channel) []string {
	return []string{
		ch.ID,
		ch.Type,
		ch.Name,
		ch.Address,
	}
}
