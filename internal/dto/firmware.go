package dto

import (
	"fmt"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/google/uuid"
)

type FirmwareReleaseResponse struct {
	Uuid         uuid.UUID `json:"uuid"`
	Version      string    `json:"version"`
	Build        string    `json:"build"`
	FileSize     int64     `json:"file_size"`
	Sha256       string    `json:"sha256"`
	ReleaseNotes string    `json:"release_notes"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewFirmwareReleaseResponse(r db.FirmwareRelease) FirmwareReleaseResponse {
	return FirmwareReleaseResponse{
		Uuid:         r.Uuid,
		Version:      r.Version,
		Build:        r.Build,
		FileSize:     r.FileSize,
		Sha256:       r.Sha256,
		ReleaseNotes: r.ReleaseNotes,
		CreatedAt:    r.CreatedAt,
	}
}

// FirmwareManifestResponse is what the controller reads before updating.
// Path is relative so the firmware can only download from the API host it
// is configured with.
type FirmwareManifestResponse struct {
	Version string `json:"version"`
	Build   string `json:"build"`
	Size    int64  `json:"size"`
	Sha256  string `json:"sha256"`
	// Base64 DER ECDSA P-256 signature of the SHA-256 digest.
	Signature string `json:"signature"`
	Path      string `json:"path"`
}

func NewFirmwareManifestResponse(r db.FirmwareRelease) FirmwareManifestResponse {
	return FirmwareManifestResponse{
		Version:   r.Version,
		Build:     r.Build,
		Size:      r.FileSize,
		Sha256:    r.Sha256,
		Signature: r.Signature,
		Path:      fmt.Sprintf("/api/v1/firmware/%s/download", r.Uuid),
	}
}

type ControllerFirmwareResponse struct {
	ControllerUuid   uuid.UUID  `json:"controller_uuid"`
	CurrentVersion   string     `json:"current_version"`
	LatestVersion    *string    `json:"latest_version"`
	ReleaseNotes     *string    `json:"release_notes"`
	UpdateAvailable  bool       `json:"update_available"`
	AutoUpdate       bool       `json:"auto_update"`
	OtaStatus        *string    `json:"ota_status"`
	OtaTargetVersion *string    `json:"ota_target_version"`
	OtaError         *string    `json:"ota_error"`
	OtaUpdatedAt     *time.Time `json:"ota_updated_at"`
	// Why "update now" is disabled: OFFLINE, IRRIGATING or IN_PROGRESS.
	BlockedReason *string `json:"blocked_reason"`
}

type SetAutoUpdateRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

type ControllerUriRequest struct {
	UUID string `uri:"uuid" binding:"required,uuid"`
}

type FirmwareUriRequest struct {
	UUID string `uri:"uuid" binding:"required,uuid"`
}

type PublishFirmwareForm struct {
	Version      string `form:"version" binding:"required,max=50"`
	Build        string `form:"build" binding:"required,max=50"`
	ReleaseNotes string `form:"release_notes" binding:"max=5000"`
}
