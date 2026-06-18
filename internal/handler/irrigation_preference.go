package handler

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateIrrigationPreference godoc
// @Summary      Create an irrigation preference
// @Description  Creates a new irrigation preference configuration for a specific device based on its UUID.
// @Tags         Irrigation Preferences
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateIrrigationPreferenceRequest true "Irrigation Preference Request Body"
// @Success      201  {object}  dto.IrrigationPreferenceResponse
// @Failure      400  {object}  object  "Bad Request - Invalid input or UUID"
// @Failure      404  {object}  object  "Not Found - Device not found"
// @Failure      500  {object}  object  "Internal Server Error - Database or server issues"
// @Router       /irrigation-preferences [post]
func (server *Server) CreateIrrigationPreference(ctx *gin.Context) {
	var req dto.CreateIrrigationPreferenceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	deviceUUID, err := uuid.Parse(req.DeviceUUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	device, err := server.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	var startHour sql.NullTime
	if req.StartHour != nil {
		t, err := time.Parse("15:04", *req.StartHour)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}

		startHour = sql.NullTime{
			Time:  t,
			Valid: true,
		}
	}

	var endHour sql.NullTime
	if req.EndHour != nil {
		t, err := time.Parse("15:04", *req.EndHour)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		}

		endHour = sql.NullTime{
			Time:  t,
			Valid: true,
		}
	}

	arg := db.CreateIrrigationPreferencesParams{
		DeviceID:             device.ID,
		IrrigationMode:       req.IrrigationMode,
		MoistureThreshold:    req.MoistureThreshold,
		DryTimeMinutes:       req.DryTimeMinutes,
		MaxIrrigationsPerDay: req.MaxIrrigationsPerDay,
		StartHour:            startHour,
		EndHour:              endHour,
	}

	irrigationPreference, err := server.store.CreateIrrigationPreferences(ctx, arg)
	if err != nil {
		log.Printf("erro ao criar irrigation preference: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewIrrigationPreferenceResponse(irrigationPreference))
}

// GetIrrigationPreference godoc
// @Summary      Get an irrigation preference by ID
// @Description  Retrieves the details of a specific irrigation preference using its ID.
// @Tags         Irrigation Preferences
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Irrigation Preference ID" format(uuid)
// @Success      200  {object}  dto.IrrigationPreferenceResponse
// @Failure      400  {object}  object  "Bad Request - Invalid ID format"
// @Failure      404  {object}  object  "Not Found - Irrigation preference not found"
// @Failure      500  {object}  object  "Internal Server Error - Database or server issues"
// @Router       /irrigation-preferences/{id} [get]
func (server *Server) GetIrrigationPreference(ctx *gin.Context) {
	var req dto.GetIrrigationPreferenceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	irrigationPreferenceUUID, err := uuid.Parse(req.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	irrigationPreference, err := server.store.GetIrrigationPreference(ctx, irrigationPreferenceUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewIrrigationPreferenceResponse(irrigationPreference))
}

// ListDeviceIrrigationPreferences godoc
// @Summary      Get irrigation preferences by Device UUID
// @Description  Retrieves all irrigation preferences associated with a specific device using the device's UUID.
// @Tags         Irrigation Preferences
// @Accept       json
// @Produce      json
// @Param        device_id  path      string  true  "Device UUID" format(uuid)
// @Success      200        {array}   dto.IrrigationPreferenceResponse
// @Failure      400        {object}  object  "Bad Request - Invalid Device UUID format"
// @Failure      404        {object}  object  "Not Found - Device or preferences not found"
// @Failure      500        {object}  object  "Internal Server Error - Database or server issues"
// @Router       /devices/{device_id}/irrigation-preferences [get]
func (server *Server) GetIrrigationPreferenceByDevice(ctx *gin.Context) {
	var req dto.GetIrrigationPreferenceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	deviceUUID, err := uuid.Parse(req.UUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	device, err := server.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	irrigationPreference, err := server.store.GetIrrigationPreferenceByDevice(ctx, device.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewIrrigationPreferenceResponse(irrigationPreference))
}
