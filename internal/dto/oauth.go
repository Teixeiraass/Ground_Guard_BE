package dto

type OAuthLoginRequest struct {
	IDToken     string `json:"id_token" binding:"required"`
	DeviceModel string `json:"device_model"`
	Platform    string `json:"platform" binding:"omitempty,oneof=android ios"`
	OsVersion   string `json:"os_version"`
	AppVersion  string `json:"app_version"`
}
