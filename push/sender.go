package push

import "context"

// Sender abstracts push notification delivery for device tokens.
type Sender interface {
	// SendToUser sends the same title/body to every token and returns the
	// subset of tokens that are no longer valid (e.g. uninstalled app),
	// so the caller can delete them.
	SendToUser(ctx context.Context, tokens []string, title, body string) (invalidTokens []string, err error)
}
