package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
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
	UserImage         *string    `json:"user_image"`
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
        UserImage:         imagePath, 
        PasswordChangedAt: user.PasswordChangedAt,
        CreatedAt:         user.CreatedAt,
    }
}

type LoginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginUserResponse struct {
	AccessToken          string       `json:"access_token"`
	RefreshToken         string       `json:"refresh_token"`
	AccessTokenExpiresAt time.Time    `json:"access_token_expires_at"`
	User                 UserResponse `json:"user"`
}

type RenewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RenewAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type UpdateUserNameRequest struct {
	FullName string `json:"full_name" binding:"required"`
}
