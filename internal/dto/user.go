package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type UserResponse struct {
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	EmailVerified     bool      `json:"email_verified"`
	UserImage         *string   `json:"user_image"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func NewUserResponse(user db.User) UserResponse {
	var imagePath *string
	if user.ProfileImage.Valid && user.ProfileImage.String != "" {
		path := user.ProfileImage.String
		imagePath = &path
	}

	return UserResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		EmailVerified:     user.EmailVerified,
		UserImage:         imagePath,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}
}

type LoginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	// Campos opcionais de telemetria do dispositivo, enviados pelo app. Apps antigos
	// que ainda não mandam esses campos continuam logando normalmente (ficam vazios).
	DeviceModel string `json:"device_model"`
	Platform    string `json:"platform" binding:"omitempty,oneof=android ios"`
	OsVersion   string `json:"os_version"`
	AppVersion  string `json:"app_version"`
}

type LoginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"acces_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  UserResponse `json:"user"`
}

type RenewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RenewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"acces_token_expires_at"`
}

type LogoutUserRequest struct {
	SessionID uuid.UUID `json:"session_id" binding:"required"`
}

type UpdateUserNameRequest struct {
	FullName string `json:"full_name" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}
