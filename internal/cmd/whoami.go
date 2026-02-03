package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dedene/frontapp-cli/internal/auth"
	"github.com/dedene/frontapp-cli/internal/errfmt"
	"github.com/dedene/frontapp-cli/internal/output"
)

type WhoamiCmd struct{}

func (c *WhoamiCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	client, err := getClient(flags)
	if err != nil {
		return err
	}

	mode, err := resolveOutputMode(flags)
	if err != nil {
		return err
	}

	// Get account info
	me, err := client.Me(ctx)
	if err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(err))

		return err
	}

	// Try to find authenticated teammate
	var teammate *struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		IsAdmin   bool   `json:"is_admin"`
	}

	// Get stored email from auth
	storedEmail, _ := auth.GetAuthenticatedEmail(flags.Client)

	// Try to find matching teammate
	teammates, err := client.ListTeammates(ctx)
	if err == nil {
		for _, t := range teammates.Results {
			if t.Email == storedEmail {
				teammate = &struct {
					ID        string `json:"id"`
					Email     string `json:"email"`
					Username  string `json:"username"`
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
					IsAdmin   bool   `json:"is_admin"`
				}{
					ID:        t.ID,
					Email:     t.Email,
					Username:  t.Username,
					FirstName: t.FirstName,
					LastName:  t.LastName,
					IsAdmin:   t.IsAdmin,
				}

				break
			}
		}
	}

	if mode.JSON {
		result := map[string]any{
			"account": me,
		}
		if teammate != nil {
			result["teammate"] = teammate
		}

		return output.WriteJSON(os.Stdout, result)
	}

	// Show account info
	fmt.Fprintf(os.Stdout, "Account:   %s\n", me.ID)

	// Show teammate info if found
	if teammate != nil {
		fmt.Fprintf(os.Stdout, "Teammate:  %s\n", teammate.ID)
		fmt.Fprintf(os.Stdout, "Email:     %s\n", teammate.Email)
		fmt.Fprintf(os.Stdout, "Username:  %s\n", teammate.Username)
		fmt.Fprintf(os.Stdout, "Name:      %s %s\n", teammate.FirstName, teammate.LastName)
		fmt.Fprintf(os.Stdout, "Admin:     %v\n", teammate.IsAdmin)
	} else if storedEmail != "" {
		fmt.Fprintf(os.Stdout, "Email:     %s (stored)\n", storedEmail)
	}

	return nil
}
