package push

import "context"

type noopSender struct{}

// NewNoopSender returns a Sender that does nothing, used when push
// notifications aren't configured (no Firebase credentials file).
func NewNoopSender() Sender {
	return &noopSender{}
}

func (s *noopSender) SendToUser(_ context.Context, _ []string, _, _ string) ([]string, error) {
	return nil, nil
}
