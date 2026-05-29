package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQrToken(t *testing.T) {
	qrtoken, err := GenerateQRToken(8)
	require.NoError(t, err)
	require.NotEmpty(t, qrtoken)
}