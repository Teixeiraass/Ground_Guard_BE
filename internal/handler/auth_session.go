package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

func (server *Server) createLoginResponse(ctx context.Context, user db.User, userAgent, clientIP string, device deviceInfo) (dto.LoginUserResponse, error) {
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

	if device.DeviceModel != "" || device.Platform != "" || device.OsVersion != "" || device.AppVersion != "" {
		_, _ = server.store.UpdateUserDeviceInfo(ctx, db.UpdateUserDeviceInfoParams{
			Uuid:            user.Uuid,
			LastDeviceModel: sql.NullString{String: device.DeviceModel, Valid: device.DeviceModel != ""},
			LastPlatform:    sql.NullString{String: device.Platform, Valid: device.Platform != ""},
			LastOsVersion:   sql.NullString{String: device.OsVersion, Valid: device.OsVersion != ""},
			LastAppVersion:  sql.NullString{String: device.AppVersion, Valid: device.AppVersion != ""},
		})
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

	// Linking by email is only safe when the provider vouches for it.
	if !identity.EmailVerified {
		return db.User{}, ErrOAuthEmailNotVerified
	}
	identity.Email = service.NormalizeEmail(identity.Email)

	user, err := server.store.GetUserByEmail(ctx, identity.Email)
	if err != nil && err != sql.ErrNoRows {
		return db.User{}, err
	}

	// Pre-hijack defense: someone could have signed up with this email and a
	// password of their choosing without ever verifying it. Now that the
	// provider proved who owns the address, lock that password out.
	if err == nil && !user.EmailVerified {
		if err := server.revokeUnverifiedCredentials(ctx, user); err != nil {
			return db.User{}, err
		}
	}

	if err == sql.ErrNoRows {
		hashedPassword, err := util.HashPassword(secureRandomString())
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

	// O provedor OAuth (Google/Apple) já verificou esse email antes de emitir o
	// id_token, então não faz sentido exigir uma segunda verificação nossa.
	if identity.EmailVerified && !user.EmailVerified {
		if verifiedUser, markErr := server.store.MarkUserEmailVerified(ctx, user.Uuid); markErr == nil {
			user = verifiedUser
		}
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

// deviceInfo carrega os metadados opcionais de dispositivo/app enviados pelo
// cliente no login, pra gravar em users.last_*.
type deviceInfo struct {
	DeviceModel string
	Platform    string
	OsVersion   string
	AppVersion  string
}

var ErrOAuthEmailNotVerified = errors.New("the provider has not verified this email")

func (server *Server) revokeUnverifiedCredentials(ctx *gin.Context, user db.User) error {
	hashed, err := util.HashPassword(secureRandomString())
	if err != nil {
		return err
	}
	if _, err := server.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		Uuid:           user.Uuid,
		HashedPassword: hashed,
	}); err != nil {
		return err
	}
	return server.store.BlockAllSessionsForUser(ctx, user.ID)
}

// secureRandomString is for credentials; util.RandomString uses math/rand.
func secureRandomString() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
