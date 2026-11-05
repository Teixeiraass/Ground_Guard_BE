package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyTurnstileSkipsWhenNotConfigured(t *testing.T) {
	ok, err := VerifyTurnstile(context.Background(), "", "any-token", "127.0.0.1")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyTurnstileRejectsEmptyResponseWhenConfigured(t *testing.T) {
	ok, err := VerifyTurnstile(context.Background(), "secret-key", "", "127.0.0.1")
	require.NoError(t, err)
	require.False(t, ok)
}
