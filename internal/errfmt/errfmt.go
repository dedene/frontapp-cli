package errfmt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dedene/frontapp-cli/internal/api"
	"github.com/dedene/frontapp-cli/internal/auth"
)

// Format formats an error into a user-friendly message with actionable suggestions.
func Format(err error) string {
	if err == nil {
		return ""
	}

	var wrongTypeErr *api.WrongResourceTypeError
	if errors.As(err, &wrongTypeErr) {
		return formatWrongResourceTypeError(wrongTypeErr)
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return formatAPIError(apiErr)
	}

	var authErr *api.AuthError
	if errors.As(err, &authErr) {
		return formatAuthError(authErr)
	}

	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return formatRateLimitError(rateLimitErr)
	}

	var circuitBreakerErr *api.CircuitBreakerError
	if errors.As(err, &circuitBreakerErr) {
		return formatCircuitBreakerError()
	}

	if errors.Is(err, auth.ErrNotAuthenticated) {
		return formatNotAuthenticatedError()
	}

	return fmt.Sprintf("Error: %v", err)
}

func formatAPIError(err *api.APIError) string {
	var sb strings.Builder

	switch err.StatusCode {
	case 401:
		sb.WriteString("Error: Not authenticated\n\n")
		sb.WriteString("  Run 'frontcli auth login' to authenticate with Front.\n")

	case 403:
		sb.WriteString("Error: Access denied (403)\n\n")
		sb.WriteString("  You don't have permission to perform this action.\n")
		sb.WriteString("  Check your account permissions in Front.\n")

	case 404:
		sb.WriteString("Error: Not found (404)\n\n")

		if err.Details != "" {
			sb.WriteString("  " + err.Details + "\n\n")
		}

		// Check if wrong ID type was used
		if err.RequestedID != "" && err.ExpectedResource != "" {
			if hint := getWrongIDTypeHint(err.RequestedID, err.ExpectedResource); hint != "" {
				sb.WriteString(hint)

				return sb.String()
			}
		}

		sb.WriteString("  The resource doesn't exist or you don't have access.\n")

	case 429:
		sb.WriteString("Error: Rate limit exceeded (429)\n\n")
		sb.WriteString("  You've hit Front's API rate limit.\n")
		sb.WriteString("  Tip: Use --limit flag to reduce result set size.\n")

	default:
		sb.WriteString(fmt.Sprintf("Error: %s (%d)\n", err.Message, err.StatusCode))

		if err.Details != "" {
			sb.WriteString("\n  " + err.Details + "\n")
		}
	}

	return sb.String()
}

func formatAuthError(err *api.AuthError) string {
	var sb strings.Builder

	sb.WriteString("Error: Authentication failed\n\n")
	sb.WriteString(fmt.Sprintf("  %v\n\n", err.Err))
	sb.WriteString("  Try running 'frontcli auth login' to re-authenticate.\n")

	return sb.String()
}

func formatRateLimitError(err *api.RateLimitError) string {
	var sb strings.Builder

	sb.WriteString("Error: Rate limit exceeded\n\n")

	if err.RetryAfter > 0 {
		sb.WriteString(fmt.Sprintf("  Retry after %d seconds.\n", err.RetryAfter))
	}

	sb.WriteString("  Tip: Use --limit flag to reduce result set size.\n")

	return sb.String()
}

func formatCircuitBreakerError() string {
	var sb strings.Builder

	sb.WriteString("Error: Service temporarily unavailable\n\n")
	sb.WriteString("  Too many consecutive failures. Please wait and try again.\n")

	return sb.String()
}

func formatNotAuthenticatedError() string {
	var sb strings.Builder

	sb.WriteString("Error: Not authenticated\n\n")
	sb.WriteString("  Run 'frontcli auth login' to authenticate with Front.\n\n")
	sb.WriteString("  If you need to set up OAuth credentials first:\n")
	sb.WriteString("    frontcli auth setup <client_id>\n")

	return sb.String()
}

func formatWrongResourceTypeError(err *api.WrongResourceTypeError) string {
	var sb strings.Builder

	sb.WriteString("Error: Wrong ID type\n\n")
	sb.WriteString(fmt.Sprintf("  '%s' is a %s ID, but a %s ID was expected.\n\n", err.ID, err.ActualType, err.ExpectedType))

	// Suggest the correct command based on the actual resource type
	if suggestion := getSuggestionForResource(err.ActualType, err.ID); suggestion != "" {
		sb.WriteString(fmt.Sprintf("  Try: %s\n", suggestion))
	}

	return sb.String()
}

// getWrongIDTypeHint returns a hint if the ID has a wrong prefix for the expected resource.
func getWrongIDTypeHint(id, expectedResource string) string {
	actualType := api.GetResourceType(id)
	if actualType == "" || actualType == expectedResource {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("  '%s' is a %s ID, but a %s ID was expected.\n\n", id, actualType, expectedResource))

	if suggestion := getSuggestionForResource(actualType, id); suggestion != "" {
		sb.WriteString(fmt.Sprintf("  Try: %s\n", suggestion))
	}

	return sb.String()
}

// getSuggestionForResource returns a CLI command suggestion for accessing a resource.
func getSuggestionForResource(resourceType, id string) string {
	switch resourceType {
	case "conversation":
		return fmt.Sprintf("frontcli conv get %s", id)
	case "message":
		return fmt.Sprintf("frontcli messages get %s", id)
	case "comment":
		return fmt.Sprintf("frontcli comments get %s", id)
	case "contact":
		return fmt.Sprintf("frontcli contacts get %s", id)
	case "teammate":
		return fmt.Sprintf("frontcli teammates get %s", id)
	case "tag":
		return fmt.Sprintf("frontcli tags get %s", id)
	case "inbox":
		return fmt.Sprintf("frontcli inboxes get %s", id)
	case "channel":
		return fmt.Sprintf("frontcli channels get %s", id)
	default:
		return ""
	}
}
