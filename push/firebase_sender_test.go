package push_test

import (
	"context"
	"testing"

	"github.com/Teixeiraass/ground_guard_be/push"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/stretchr/testify/require"
)

func TestNewFirebaseSenderWhenDisabled(t *testing.T) {
	config := util.Config{FirebaseCredentialsFile: ""}

	sender, err := push.NewFirebaseSender(config)
	require.NoError(t, err)

	invalidTokens, err := sender.SendToUser(context.Background(), []string{"some-token"}, "title", "body")
	require.NoError(t, err)
	require.Empty(t, invalidTokens)
}
