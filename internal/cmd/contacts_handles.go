package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dedene/frontapp-cli/internal/api"
	"github.com/dedene/frontapp-cli/internal/errfmt"
	"github.com/dedene/frontapp-cli/internal/output"
)

type ContactHandlesCmd struct {
	ID string `arg:"" help:"Contact ID"`
}

func (c *ContactHandlesCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	contact, err := client.GetContact(ctx, c.ID)
	if err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, map[string]any{"handles": contact.Handles})
	}

	if len(contact.Handles) == 0 {
		fmt.Fprintln(os.Stdout, "No handles found.")

		return nil
	}

	tbl := output.NewTableWriter(os.Stdout, mode.Plain)
	tbl.AddRow("HANDLE", "SOURCE")

	for _, h := range contact.Handles {
		tbl.AddRow(h.Handle, h.Source)
	}

	return tbl.Flush()
}

type ContactHandleCmd struct {
	Add    ContactHandleAddCmd    `cmd:"" help:"Add a handle to a contact"`
	Delete ContactHandleDeleteCmd `cmd:"" help:"Delete a contact handle"`
}

type ContactHandleAddCmd struct {
	ContactID string `arg:"" help:"Contact ID"`
	Type      string `required:"" help:"Handle type (email, phone, etc)"`
	Value     string `required:"" help:"Handle value"`
}

func (c *ContactHandleAddCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	req := map[string]string{
		"source": c.Type,
		"handle": c.Value,
	}

	var result api.Handle
	if err := client.Post(ctx, fmt.Sprintf("/contacts/%s/handles", c.ContactID), req, &result); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stdout, "Handle added: %s\n", result.Handle)

	return nil
}

type ContactHandleDeleteCmd struct {
	ID string `arg:"" help:"Handle ID"`
}

func (c *ContactHandleDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if err := client.Delete(ctx, fmt.Sprintf("/contact_handles/%s", c.ID)); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintln(os.Stdout, "Handle deleted")

	return nil
}
