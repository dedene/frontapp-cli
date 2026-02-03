package cmd

import (
	"github.com/dedene/frontapp-cli/internal/api"
	"github.com/dedene/frontapp-cli/internal/auth"
	"github.com/dedene/frontapp-cli/internal/config"
)

var newClientFromAuth = api.NewClientFromAuth

// getClient creates an API client using stored auth credentials.
func getClient(flags *RootFlags) (*api.Client, error) {
	email, err := config.ResolveAccount(flags.Account)
	if err != nil {
		return nil, err
	}

	clientName := flags.Client
	if clientName == "" {
		clientName = "default"
	}

	if email == "" {
		// Try to get email from stored tokens
		email, err = auth.GetAuthenticatedEmail(clientName)
		if err != nil {
			return nil, &api.AuthError{Err: err}
		}
	}

	clientName, err = config.ResolveClientForAccount(email, flags.Client)
	if err != nil {
		return nil, err
	}

	return newClientFromAuth(clientName, email)
}
