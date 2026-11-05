package dto

type RequestPasswordResetRequest struct {
	Email          string `json:"email" binding:"required,email"`
	TurnstileToken string `json:"turnstile_token"`
}

type ConfirmPasswordResetRequest struct {
	Token          string `json:"token" binding:"required"`
	NewPassword    string `json:"new_password" binding:"required,min=6"`
	TurnstileToken string `json:"turnstile_token"`
}
