package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/mailer"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	userStatusActive  = "ACTIVE"
	userStatusBanned  = "BANNED"
	userStatusDeleted = "DELETED"

	emailVerificationTokenTTL = 24 * time.Hour
	passwordResetTokenTTL     = 1 * time.Hour
)

var (
	ErrUserBanned               = errors.New("user is banned")
	ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")
	ErrInvalidOrExpiredToken    = errors.New("invalid or expired token")
)

type UserService interface {
	CreateUser(ctx context.Context, req dto.CreateUserRequest) (*db.User, error)
	GetUser(ctx context.Context, userUUID uuid.UUID) (*db.User, error)
	LoginUser(ctx context.Context, req dto.LoginUserRequest, userAgent string, clientIp string) (*dto.LoginUserResponse, error)
	RenewAccessToken(ctx context.Context, req dto.RenewAccessTokenRequest) (*dto.RenewAccessTokenResponse, error)
	LogoutUser(ctx context.Context, sessionID uuid.UUID, userID int64) error
	UpdateUserName(ctx context.Context, userUUID uuid.UUID, fullName string) (*db.User, error)
	UpdateProfileImage(ctx context.Context, userUUID uuid.UUID, imageData []byte, username string) (*db.User, error)
	ListUsers(ctx context.Context, search string, limit, offset int32) ([]db.User, error)
	BanUser(ctx context.Context, userUUID uuid.UUID, reason string) (*db.User, error)
	UnbanUser(ctx context.Context, userUUID uuid.UUID) (*db.User, error)
	ChangePassword(ctx context.Context, userUUID uuid.UUID, currentPassword, newPassword string) error
	DeleteAccount(ctx context.Context, userUUID uuid.UUID, password string) error
	VerifyEmail(ctx context.Context, rawToken string) error
	ResendVerificationEmail(ctx context.Context, userUUID uuid.UUID) error
	RequestPasswordReset(ctx context.Context, email, turnstileToken, clientIP string) error
	ConfirmPasswordReset(ctx context.Context, rawToken, newPassword, turnstileToken, clientIP string) error
}

type userService struct {
	store      db.Store
	tokenMaker token.Maker
	config     util.Config
	mailer     *mailer.Mailer
}

func NewUserService(store db.Store, tokenMaker token.Maker, config util.Config) UserService {
	return &userService{
		store:      store,
		tokenMaker: tokenMaker,
		config:     config,
		mailer:     mailer.NewMailer(config),
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
		Email:          NormalizeEmail(req.Email),
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

	if err := s.sendEmailVerification(ctx, user); err != nil {
		// Não falha o cadastro por causa do email: a conta já foi criada,
		// o usuário pode pedir o reenvio depois pela tela de perfil.
		log.Printf("[user_service] falha ao enviar email de verificação para %s: %v", user.Email, err)
	}

	return &user, nil
}

// sendEmailVerification gera um novo token de verificação, grava hasheado no
// usuário e envia o email com o link de confirmação.
func (s *userService) sendEmailVerification(ctx context.Context, user db.User) error {
	rawToken, err := util.GenerateQRToken(32)
	if err != nil {
		return err
	}

	_, err = s.store.SetEmailVerificationToken(ctx, db.SetEmailVerificationTokenParams{
		Uuid: user.Uuid,
		EmailVerificationToken: sql.NullString{
			String: util.HashToken(rawToken),
			Valid:  true,
		},
		EmailVerificationExpiresAt: sql.NullTime{
			Time:  time.Now().Add(emailVerificationTokenTTL),
			Valid: true,
		},
	})
	if err != nil {
		return err
	}

	link := s.config.AppBaseURL + "/api/v1/verify-email?token=" + rawToken
	return s.mailer.SendVerificationEmail(user.Email, link)
}

func (s *userService) GetUser(ctx context.Context, userUUID uuid.UUID) (*db.User, error) {
	user, err := s.store.GetUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// NormalizeEmail makes email lookups case-insensitive and trims spaces.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

var (
	dummyHashOnce sync.Once
	dummyHash     string
)

// burnPasswordCheck spends the same bcrypt time as a real check, so response
// timing doesn't reveal whether an email is registered.
func burnPasswordCheck(password string) {
	dummyHashOnce.Do(func() {
		dummyHash, _ = util.HashPassword("ground-guard-timing-equalizer")
	})
	_ = util.CheckPassword(password, dummyHash)
}

func (s *userService) LoginUser(ctx context.Context, req dto.LoginUserRequest, userAgent string, clientIp string) (*dto.LoginUserResponse, error) {
	user, err := s.store.GetUserByEmail(ctx, NormalizeEmail(req.Email))
	if errors.Is(err, sql.ErrNoRows) {
		burnPasswordCheck(req.Password)
		return nil, errors.New("unauthorized")
	}
	if err != nil {
		return nil, err
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		return nil, errors.New("unauthorized")
	}

	if user.Status != userStatusActive {
		if user.Status == userStatusBanned {
			return nil, ErrUserBanned
		}
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

	s.updateDeviceInfo(ctx, user.Uuid, req.DeviceModel, req.Platform, req.OsVersion, req.AppVersion)

	return &dto.LoginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  dto.NewUserResponse(user),
	}, nil
}

// updateDeviceInfo grava o último modelo/plataforma/versão de SO/versão do app
// usados pelo usuário. Best-effort: nunca falha o login por isso, e não grava
// nada se o app não enviou nenhum campo (apps antigos, ou chamadas de teste).
func (s *userService) updateDeviceInfo(ctx context.Context, userUUID uuid.UUID, deviceModel, platform, osVersion, appVersion string) {
	if deviceModel == "" && platform == "" && osVersion == "" && appVersion == "" {
		return
	}

	_, err := s.store.UpdateUserDeviceInfo(ctx, db.UpdateUserDeviceInfoParams{
		Uuid:            userUUID,
		LastDeviceModel: sql.NullString{String: deviceModel, Valid: deviceModel != ""},
		LastPlatform:    sql.NullString{String: platform, Valid: platform != ""},
		LastOsVersion:   sql.NullString{String: osVersion, Valid: osVersion != ""},
		LastAppVersion:  sql.NullString{String: appVersion, Valid: appVersion != ""},
	})
	if err != nil {
		log.Printf("[user_service] falha ao gravar device info do usuário %s: %v", userUUID, err)
	}
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

func (s *userService) ListUsers(ctx context.Context, search string, limit, offset int32) ([]db.User, error) {
	return s.store.ListUsers(ctx, db.ListUsersParams{
		Search:     search,
		PageLimit:  limit,
		PageOffset: offset,
	})
}

func (s *userService) BanUser(ctx context.Context, userUUID uuid.UUID, reason string) (*db.User, error) {
	user, err := s.store.BanUser(ctx, db.BanUserParams{
		Uuid: userUUID,
		BanReason: sql.NullString{
			String: reason,
			Valid:  reason != "",
		},
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userService) UnbanUser(ctx context.Context, userUUID uuid.UUID) (*db.User, error) {
	user, err := s.store.UnbanUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userService) ChangePassword(ctx context.Context, userUUID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.store.GetUser(ctx, userUUID)
	if err != nil {
		return err
	}

	if err := util.CheckPassword(currentPassword, user.HashedPassword); err != nil {
		return ErrCurrentPasswordIncorrect
	}

	hashedPassword, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = s.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		Uuid:           userUUID,
		HashedPassword: hashedPassword,
	})
	return err
}

func (s *userService) DeleteAccount(ctx context.Context, userUUID uuid.UUID, password string) error {
	user, err := s.store.GetUser(ctx, userUUID)
	if err != nil {
		return err
	}

	if err := util.CheckPassword(password, user.HashedPassword); err != nil {
		return ErrCurrentPasswordIncorrect
	}

	if _, err := s.store.SoftDeleteUser(ctx, userUUID); err != nil {
		return err
	}

	if err := s.store.BlockAllSessionsForUser(ctx, user.ID); err != nil {
		log.Printf("[user_service] falha ao revogar sessões do usuário %s: %v", userUUID, err)
	}

	return nil
}

func (s *userService) VerifyEmail(ctx context.Context, rawToken string) error {
	_, err := s.store.VerifyUserEmailByToken(ctx, sql.NullString{
		String: util.HashToken(rawToken),
		Valid:  true,
	})
	if err == sql.ErrNoRows {
		return ErrInvalidOrExpiredToken
	}
	return err
}

func (s *userService) ResendVerificationEmail(ctx context.Context, userUUID uuid.UUID) error {
	user, err := s.store.GetUser(ctx, userUUID)
	if err != nil {
		return err
	}

	if user.EmailVerified {
		return nil
	}

	return s.sendEmailVerification(ctx, user)
}

// RequestPasswordReset sempre retorna nil quando o email não existe, pra não
// revelar pra quem está pedindo se aquele email tem conta ou não (mesmo
// raciocínio já usado no login, que responde igual pra email e senha errados).
func (s *userService) RequestPasswordReset(ctx context.Context, email, turnstileToken, clientIP string) error {
	ok, err := util.VerifyTurnstile(ctx, s.config.TurnstileSecretKey, turnstileToken, clientIP)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidOrExpiredToken
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	rawToken, err := util.GenerateQRToken(32)
	if err != nil {
		return err
	}

	_, err = s.store.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: util.HashToken(rawToken),
		ExpiresAt: time.Now().Add(passwordResetTokenTTL),
	})
	if err != nil {
		return err
	}

	link := s.config.AppBaseURL + "/api/v1/reset-password?token=" + rawToken
	return s.mailer.SendPasswordResetEmail(user.Email, link)
}

func (s *userService) ConfirmPasswordReset(ctx context.Context, rawToken, newPassword, turnstileToken, clientIP string) error {
	ok, err := util.VerifyTurnstile(ctx, s.config.TurnstileSecretKey, turnstileToken, clientIP)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidOrExpiredToken
	}

	resetToken, err := s.store.ConsumePasswordResetToken(ctx, util.HashToken(rawToken))
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrInvalidOrExpiredToken
		}
		return err
	}

	user, err := s.store.GetUserByID(ctx, resetToken.UserID)
	if err != nil {
		return err
	}

	hashedPassword, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if _, err := s.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		Uuid:           user.Uuid,
		HashedPassword: hashedPassword,
	}); err != nil {
		return err
	}

	// Any other link still sitting in the user's inbox dies with this one.
	if err := s.store.InvalidatePasswordResetTokensForUser(ctx, user.ID); err != nil {
		log.Printf("[user_service] falha ao invalidar outros tokens de reset do usuário %s: %v", user.Uuid, err)
	}

	if err := s.store.BlockAllSessionsForUser(ctx, user.ID); err != nil {
		log.Printf("[user_service] falha ao revogar sessões do usuário %s: %v", user.Uuid, err)
	}

	return nil
}
