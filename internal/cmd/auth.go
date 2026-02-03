package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"

	"github.com/dedene/frontapp-cli/internal/api"
	"github.com/dedene/frontapp-cli/internal/auth"
	"github.com/dedene/frontapp-cli/internal/config"
)

type AuthCmd struct {
	Setup  AuthSetupCmd  `cmd:"" help:"Configure OAuth credentials"`
	Login  AuthLoginCmd  `cmd:"" help:"Authenticate with Front"`
	Logout AuthLogoutCmd `cmd:"" help:"Remove stored tokens"`
	Status AuthStatusCmd `cmd:"" help:"Show authentication status"`
	List   AuthListCmd   `cmd:"" help:"List authenticated accounts"`
}

type AuthSetupCmd struct {
	ClientID     string `arg:"" help:"OAuth client ID"`
	ClientSecret string `name:"client-secret" help:"OAuth client secret (for non-interactive use)"`
	ClientName   string `help:"Client name (default: default)" default:"default" name:"client-name"`
	RedirectURI  string `help:"OAuth redirect URI" default:"https://localhost:8484/callback"`
}

func (c *AuthSetupCmd) Run() error {
	secret := c.ClientSecret

	if secret == "" {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Print("Client Secret: ")

			bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println() // newline after hidden input

			if err != nil {
				return fmt.Errorf("failed to read secret: %w", err)
			}

			secret = string(bytes)
		} else {
			return fmt.Errorf("client secret required: use --client-secret flag or run interactively")
		}
	}

	creds := config.OAuthCredentials{
		ClientID:     c.ClientID,
		ClientSecret: secret,
		RedirectURI:  c.RedirectURI,
	}

	if err := config.WriteClientCredentials(c.ClientName, creds); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}

	path, _ := config.ClientCredentialsPath(c.ClientName)
	fmt.Fprintf(os.Stdout, "Credentials saved to %s\n", path)
	fmt.Fprintln(os.Stdout, "Run 'frontcli auth login' to authenticate.")

	return nil
}

type AuthLoginCmd struct {
	Email        string `help:"Email/identifier to associate with this token" name:"email"`
	ClientName   string `help:"Client name" default:"default" name:"client-name"`
	ForceConsent bool   `help:"Force consent prompt even if already authorized"`
	Manual       bool   `help:"Manual authorization (paste URL instead of callback server)"`
}

func (c *AuthLoginCmd) Run(flags *RootFlags) error {
	ctx := context.Background()

	refreshToken, err := auth.Authorize(ctx, auth.AuthorizeOptions{
		Client:       c.ClientName,
		ForceConsent: c.ForceConsent,
		Manual:       c.Manual,
		Timeout:      3 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	// Use email from flag or try to fetch from /me
	email := c.Email
	if email == "" && flags != nil && flags.Account != "" {
		email = flags.Account
	}

	if email == "" {
		// Fetch real email from /me endpoint
		email, err = c.fetchEmail(ctx, refreshToken)
		if err != nil {
			// Don't fall back - require user to specify email
			return fmt.Errorf("could not determine your identity: %w\nUse --email flag to specify your email", err)
		}
	}

	tok := auth.Token{
		Email:        email,
		RefreshToken: refreshToken,
		CreatedAt:    time.Now().UTC(),
	}

	if err := store.SetToken(c.ClientName, email, tok); err != nil {
		return fmt.Errorf("store token: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Successfully authenticated as %s\n", email)

	return nil
}

func (c *AuthLoginCmd) fetchEmail(ctx context.Context, refreshToken string) (string, error) {
	// Create a temporary token source with the refresh token
	ts := auth.NewRefreshTokenSource(c.ClientName, refreshToken)
	client := api.NewClient(ts)

	// Try to get account info from /me
	me, err := client.Me(ctx)
	if err != nil {
		return "", err
	}

	// /me returns account info, not teammate info for OAuth
	// Use account ID as identifier if no email provided
	if me.Email != "" {
		return me.Email, nil
	}

	// For OAuth tokens, try to find the authenticated user from teammates list
	teammates, err := client.ListTeammates(ctx)
	if err != nil {
		// Fall back to account ID
		if me.ID != "" {
			return me.ID, nil
		}

		return "", fmt.Errorf("could not determine account identity")
	}

	if len(teammates.Results) == 1 {
		// Single teammate account - use that email
		return teammates.Results[0].Email, nil
	}

	if len(teammates.Results) > 1 {
		// Multiple teammates - show list and ask user to re-run with --email
		fmt.Fprintln(os.Stderr, "Multiple teammates found. Please re-run with --email flag:")
		for _, t := range teammates.Results {
			fmt.Fprintf(os.Stderr, "  - %s (%s %s)\n", t.Email, t.FirstName, t.LastName)
		}

		return "", fmt.Errorf("multiple teammates - specify --email")
	}

	// Fall back to account ID
	if me.ID != "" {
		return me.ID, nil
	}

	return "", fmt.Errorf("could not determine account identity")
}

type AuthLogoutCmd struct {
	Email      string `help:"Email/account to log out" name:"email"`
	ClientName string `help:"Client name" default:"default" name:"client-name"`
	All        bool   `help:"Log out all accounts for this client"`
}

func (c *AuthLogoutCmd) Run() error {
	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	if c.All {
		tokens, err := store.ListTokens()
		if err != nil {
			return fmt.Errorf("list tokens: %w", err)
		}

		normalizedClient, err := config.NormalizeClientNameOrDefault(c.ClientName)
		if err != nil {
			return err
		}

		count := 0

		for _, tok := range tokens {
			if tok.Client == normalizedClient {
				if err := store.DeleteToken(tok.Client, tok.Email); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove token for %s: %v\n", tok.Email, err)
				} else {
					count++
				}
			}
		}

		fmt.Fprintf(os.Stdout, "Logged out %d account(s)\n", count)

		return nil
	}

	if c.Email == "" {
		// Try to find the only account
		tokens, err := store.ListTokens()
		if err != nil {
			return fmt.Errorf("list tokens: %w", err)
		}

		normalizedClient, err := config.NormalizeClientNameOrDefault(c.ClientName)
		if err != nil {
			return err
		}

		var match auth.Token
		count := 0

		for _, tok := range tokens {
			if tok.Client == normalizedClient {
				match = tok
				count++
			}
		}

		if count == 0 {
			return fmt.Errorf("no authenticated accounts found")
		}

		if count > 1 {
			return fmt.Errorf("multiple accounts found; specify --email or use --all")
		}

		c.Email = match.Email
	}

	if err := store.DeleteToken(c.ClientName, c.Email); err != nil {
		return fmt.Errorf("remove token: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Logged out %s\n", c.Email)

	return nil
}

type AuthStatusCmd struct {
	ClientName string `help:"Client name" default:"default" name:"client-name"`
}

func (c *AuthStatusCmd) Run() error {
	// Check if credentials exist
	exists, err := config.ClientCredentialsExists(c.ClientName)
	if err != nil {
		return err
	}

	if !exists {
		fmt.Fprintln(os.Stdout, "Not configured")
		fmt.Fprintln(os.Stdout, "Run 'frontcli auth setup <client_id>' to configure.")

		return nil
	}

	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return fmt.Errorf("list tokens: %w", err)
	}

	normalizedClient, err := config.NormalizeClientNameOrDefault(c.ClientName)
	if err != nil {
		return err
	}

	count := 0

	for _, tok := range tokens {
		if tok.Client == normalizedClient {
			count++
		}
	}

	if count == 0 {
		fmt.Fprintln(os.Stdout, "OAuth credentials configured but not authenticated.")
		fmt.Fprintln(os.Stdout, "Run 'frontcli auth login' to authenticate.")

		return nil
	}

	fmt.Fprintf(os.Stdout, "Authenticated: %d account(s)\n", count)

	for _, tok := range tokens {
		if tok.Client == normalizedClient {
			fmt.Fprintf(os.Stdout, "  - %s (since %s)\n", tok.Email, tok.CreatedAt.Format("2006-01-02"))
		}
	}

	return nil
}

type AuthListCmd struct{}

func (c *AuthListCmd) Run() error {
	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return fmt.Errorf("list tokens: %w", err)
	}

	if len(tokens) == 0 {
		fmt.Fprintln(os.Stdout, "No authenticated accounts.")

		return nil
	}

	fmt.Fprintln(os.Stdout, "Authenticated accounts:")

	for _, tok := range tokens {
		fmt.Fprintf(os.Stdout, "  %s (client: %s, since %s)\n",
			tok.Email, tok.Client, tok.CreatedAt.Format("2006-01-02"))
	}

	return nil
}
