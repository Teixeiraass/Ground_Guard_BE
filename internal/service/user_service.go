package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type UserService interface {
	CreateUser(ctx context.Context, req dto.CreateUserRequest) (*db.User, error)
	GetUser(ctx context.Context, userUUID uuid.UUID) (*db.User, error)
	LoginUser(ctx context.Context, req dto.LoginUserRequest, userAgent string, clientIp string) (*dto.LoginUserResponse, error)
	RenewAccessToken(ctx context.Context, req dto.RenewAccessTokenRequest) (*dto.RenewAccessTokenResponse, error)
	LogoutUser(ctx context.Context, sessionID uuid.UUID, userID int64) error
	UpdateUserName(ctx context.Context, userUUID uuid.UUID, fullName string) (*db.User, error)
	UpdateProfileImage(ctx context.Context, userUUID uuid.UUID, imageData []byte, username string) (*db.User, error)
}

type userService struct {
	store      db.Store
	tokenMaker token.Maker
	config     util.Config
}

func NewUserService(store db.Store, tokenMaker token.Maker, config util.Config) UserService {
	return &userService{
		store:      store,
		tokenMaker: tokenMaker,
		config:     config,
	}
}

func (s *userService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*db.User, error) {
	hashPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := s.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return nil, errors.New("unique_violation")
			}
		}
		return nil, err
	}

	return &user, nil
}

func (s *userService) GetUser(ctx context.Context, userUUID uuid.UUID) (*db.User, error) {
	user, err := s.store.GetUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userService) LoginUser(ctx context.Context, req dto.LoginUserRequest, userAgent string, clientIp string) (*dto.LoginUserResponse, error) {
	user, err := s.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		return nil, errors.New("unauthorized")
	}

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(
		user.Username,
		user.ID,
		user.Uuid,
		s.config.AccessTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshPayload, err := s.tokenMaker.CreateToken(
		user.Username,
		user.ID,
		user.Uuid,
		s.config.RefreshTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	session, err := s.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIp,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		return nil, err
	}

	return &dto.LoginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  dto.NewUserResponse(user),
	}, nil
}

func (s *userService) RenewAccessToken(ctx context.Context, req dto.RenewAccessTokenRequest) (*dto.RenewAccessTokenResponse, error) {
	refreshPayload, err := s.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("unauthorized")
	}

	session, err := s.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		return nil, err
	}

	if session.IsBlocked {
		return nil, errors.New("blocked session")
	}

	if session.UserID != refreshPayload.UserID {
		return nil, errors.New("incorrect session user")
	}

	if session.RefreshToken != req.RefreshToken {
		return nil, errors.New("mismatched session token")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("expired session")
	}

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(
		refreshPayload.Username,
		refreshPayload.UserID,
		refreshPayload.UserUUID,
		s.config.AccessTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	return &dto.RenewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}, nil
}

func (s *userService) LogoutUser(ctx context.Context, sessionID uuid.UUID, userID int64) error {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.UserID != userID {
		return errors.New("invalid session")
	}

	return s.store.BlockSession(ctx, sessionID)
}

func (s *userService) UpdateUserName(ctx context.Context, userUUID uuid.UUID, fullName string) (*db.User, error) {
	arg := db.UpdateUserNameParams{
		Uuid:     userUUID,
		FullName: fullName,
	}
	user, err := s.store.UpdateUserName(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userService) UpdateProfileImage(ctx context.Context, userUUID uuid.UUID, imageData []byte, username string) (*db.User, error) {
	imagePath, err := util.SaveUserImage(imageData, username)
	if err != nil {
		return nil, err
	}

	arg := db.UpdateUserProfileImageParams{
		Uuid: userUUID,
		ProfileImage: sql.NullString{
			String: imagePath,
			Valid:  true,
		},
	}

	user, err := s.store.UpdateUserProfileImage(ctx, arg)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
