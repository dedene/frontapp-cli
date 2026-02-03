package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dedene/frontapp-cli/internal/api"
	"github.com/dedene/frontapp-cli/internal/errfmt"
	"github.com/dedene/frontapp-cli/internal/output"
)

type ContactCreateCmd struct {
	Handle      string `required:"" help:"Contact handle (e.g., email:user@example.com)"`
	Name        string `help:"Contact name"`
	Description string `help:"Contact description"`
}

func (c *ContactCreateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	// Parse handle format "type:value" or "type/value"
	handleType := "email"
	handleValue := c.Handle

	for _, sep := range []string{":", "/"} {
		if idx := strings.Index(c.Handle, sep); idx > 0 {
			handleType = c.Handle[:idx]
			handleValue = c.Handle[idx+1:]

			break
		}
	}

	req := map[string]any{
		"handles": []map[string]string{
			{"handle": handleValue, "source": handleType},
		},
	}

	if c.Name != "" {
		req["name"] = c.Name
	}

	if c.Description != "" {
		req["description"] = c.Description
	}

	var result api.Contact
	if err := client.Post(ctx, "/contacts", req, &result); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stdout, "Contact created: %s\n", result.ID)

	return nil
}

type ContactUpdateCmd struct {
	ID          string `arg:"" help:"Contact ID"`
	Name        string `help:"New name"`
	Description string `help:"New description"`
}

func (c *ContactUpdateCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	req := map[string]any{}

	if c.Name != "" {
		req["name"] = c.Name
	}

	if c.Description != "" {
		req["description"] = c.Description
	}

	if len(req) == 0 {
		return fmt.Errorf("no updates specified")
	}

	var result api.Contact
	if err := client.Patch(ctx, "/contacts/"+c.ID, req, &result); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	if mode.JSON {
		return output.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stdout, "Contact updated: %s\n", result.Name)

	return nil
}

type ContactDeleteCmd struct {
	ID string `arg:"" help:"Contact ID"`
}

func (c *ContactDeleteCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	if err := client.Delete(ctx, "/contacts/"+c.ID); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintln(os.Stdout, "Contact deleted")

	return nil
}

type ContactMergeCmd struct {
	Source string `arg:"" help:"Source contact ID (will be merged into target)"`
	Target string `arg:"" help:"Target contact ID (will receive merged data)"`
}

func (c *ContactMergeCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	req := map[string]any{
		"contact_ids":       []string{c.Source},
		"target_contact_id": c.Target,
	}

	if err := client.Post(ctx, "/contacts/merge", req, nil); err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	fmt.Fprintf(os.Stdout, "Merged %s into %s\n", c.Source, c.Target)

	return nil
}
