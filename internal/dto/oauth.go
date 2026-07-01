package dto

type OAuthLoginRequest struct {
    IDToken string `json:"id_token" binding:"required"`
}