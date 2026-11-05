package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeaderKey  = "authorization"
	AuthorizationTypeBearer = "bearer"
	AuthorizationPayloadKey = "authorization_payload"
	CurrentUserKey          = "current_user"

	UserStatusActive  = "ACTIVE"
	UserStatusBanned  = "BANNED"
	UserStatusDeleted = "DELETED"
)

func Auth(tokenMaker token.Maker, store db.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(AuthorizationHeaderKey)
		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != AuthorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		user, err := store.GetUser(context.Background(), payload.UserUUID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		if user.Status != UserStatusActive {
			switch user.Status {
			case UserStatusBanned:
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":  "sua conta foi suspensa",
					"code":   "USER_BANNED",
					"reason": user.BanReason.String,
				})
			case UserStatusDeleted:
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "sua conta foi excluída",
					"code":  "USER_DELETED",
				})
			default:
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "sua conta não está ativa",
					"code":  "USER_" + user.Status,
				})
			}
			return
		}

		// A password change or reset kills every token issued before it,
		// so a stolen token doesn't outlive the password the victim changed.
		if payload.IssuedAt.Add(time.Second).Before(user.PasswordChangedAt) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "session expired, please log in again",
				"code":  "TOKEN_REVOKED",
			})
			return
		}

		ctx.Set(AuthorizationPayloadKey, payload)
		ctx.Set(CurrentUserKey, user)
		ctx.Next()
	}
}

// RequireAdmin exige que o usuário autenticado (já carregado pelo Auth,
// que roda antes na mesma cadeia) tenha is_admin = true.
func RequireAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		value, exists := ctx.Get(CurrentUserKey)
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(errors.New("forbidden")))
			return
		}

		user, ok := value.(db.User)
		if !ok || !user.IsAdmin {
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(errors.New("admin access required")))
			return
		}

		ctx.Next()
	}
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
