package dto

import (
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type ListUsersRequest struct {
	PageID   int32  `form:"page_id" binding:"required,min=1"`
	PageSize int32  `form:"page_size" binding:"required,min=5,max=100"`
	Search   string `form:"search"`
}

type BanUserUriRequest struct {
	UUID string `uri:"uuid" binding:"required"`
}

type BanUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type AdminUserResponse struct {
	Uuid      uuid.UUID `json:"uuid"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	BanReason *string   `json:"ban_reason"`
	BannedAt  *time.Time `json:"banned_at"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

func NewAdminUserResponse(user db.User) AdminUserResponse {
	var banReason *string
	if user.BanReason.Valid {
		banReason = &user.BanReason.String
	}

	var bannedAt *time.Time
	if user.BannedAt.Valid {
		bannedAt = &user.BannedAt.Time
	}

	return AdminUserResponse{
		Uuid:      user.Uuid,
		Username:  user.Username,
		FullName:  user.FullName,
		Email:     user.Email,
		Status:    user.Status,
		BanReason: banReason,
		BannedAt:  bannedAt,
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt,
	}
}
