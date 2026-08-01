package handler

import (
	"context"
	"database/sql"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

func (server *Server) createLoginResponse(ctx context.Context, user db.User, userAgent, clientIP string) (dto.LoginUserResponse, error) {
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		user.ID,
		user.Uuid,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		return dto.LoginUserResponse{}, err
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		user.ID,
		user.Uuid,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		return dto.LoginUserResponse{}, err
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIP,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		return dto.LoginUserResponse{}, err
	}

	return dto.LoginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  dto.NewUserResponse(user),
	}, nil
}

func (server *Server) createOAuthUser(ctx *gin.Context, identity oauthIdentity) (db.User, error) {
	oauthIdentityRow, err := server.store.GetOAuthIdentityByProviderAndSubject(ctx, db.GetOAuthIdentityByProviderAndSubjectParams{
		Provider:        identity.Provider,
		ProviderSubject: identity.Subject,
	})
	if err == nil {
		return server.store.GetUserByID(ctx, oauthIdentityRow.UserID)
	}

	if err != sql.ErrNoRows {
		return db.User{}, err
	}

	user, err := server.store.GetUserByEmail(ctx, identity.Email)
	if err != nil && err != sql.ErrNoRows {
		return db.User{}, err
	}

	if err == sql.ErrNoRows {
		hashedPassword, err := util.HashPassword(util.RandomString(32))
		if err != nil {
			return db.User{}, err
		}

		user, err = server.store.CreateUser(ctx, db.CreateUserParams{
			Username:       identity.Username,
			HashedPassword: hashedPassword,
			FullName:       identity.FullName,
			Email:          identity.Email,
			ProfileImage: sql.NullString{
				String: identity.Picture,
				Valid:  identity.Picture != "",
			},
		})
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
				user, err = server.store.GetUserByEmail(ctx, identity.Email)
				if err != nil {
					return db.User{}, err
				}
			} else {
				return db.User{}, err
			}
		}
	}

	if err := server.linkOAuthIdentity(ctx, user, identity); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			oauthIdentityRow, lookupErr := server.store.GetOAuthIdentityByProviderAndSubject(ctx, db.GetOAuthIdentityByProviderAndSubjectParams{
				Provider:        identity.Provider,
				ProviderSubject: identity.Subject,
			})
			if lookupErr != nil {
				return db.User{}, lookupErr
			}

			return server.store.GetUserByID(ctx, oauthIdentityRow.UserID)
		}

		return db.User{}, err
	}

	return user, nil
}

func (server *Server) linkOAuthIdentity(ctx *gin.Context, user db.User, identity oauthIdentity) error {
	_, err := server.store.CreateOAuthIdentity(ctx, db.CreateOAuthIdentityParams{
		UserID:          user.ID,
		Provider:        identity.Provider,
		ProviderSubject: identity.Subject,
		Email:           identity.Email,
		EmailVerified:   identity.EmailVerified,
	})
	return err
}

type oauthIdentity struct {
	Provider      string
	Subject       string
	Email         string
	FullName      string
	Username      string
	EmailVerified bool
	Picture       string
}
