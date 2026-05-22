package middleware

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func AddAuthorization(
	t *testing.T,
	request *http.Request,
	tokenMaker token.Maker,
	authorizationType string,
	username string,
	userID int64,
	userUUID uuid.UUID,
	duration time.Duration,
) {
	accessToken, err := tokenMaker.CreateToken(username, userID, userUUID, duration)
	require.NoError(t, err)

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, accessToken)
	request.Header.Set(AuthorizationHeaderKey, authorizationHeader)
}
