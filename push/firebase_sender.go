package push

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/Teixeiraass/ground_guard_be/util"
	"google.golang.org/api/option"
)

type firebaseSender struct {
	client *messaging.Client
}

// NewFirebaseSender returns a real Firebase Cloud Messaging sender, or a
// no-op sender when FIREBASE_CREDENTIALS_FILE isn't configured — mirrors
// mqtt/client.NewPahoClient's gating on MQTTEnabled.
func NewFirebaseSender(config util.Config) (Sender, error) {
	if config.FirebaseCredentialsFile == "" {
		return NewNoopSender(), nil
	}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(config.FirebaseCredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("cannot initialize firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot create firebase messaging client: %w", err)
	}

	return &firebaseSender{client: client}, nil
}

func (s *firebaseSender) SendToUser(ctx context.Context, tokens []string, title, body string) ([]string, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	response, err := s.client.SendEachForMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("firebase send failed: %w", err)
	}

	var invalidTokens []string
	for i, resp := range response.Responses {
		if resp.Success {
			continue
		}
		if messaging.IsRegistrationTokenNotRegistered(resp.Error) {
			invalidTokens = append(invalidTokens, tokens[i])
		}
	}

	return invalidTokens, nil
}
