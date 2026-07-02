package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateIrrigationCommand
// @Summary      Criar comando de irrigação
// @Description  Registra um comando pendente para ser enviado ao ESP32 via MQTT.
// @Tags         irrigation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.CreateIrrigationCommandRequest  true  "Dados do comando"
// @Success      201      {object}  dto.CreateIrrigationCommandResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request"
// @Failure      401      {object}  map[string]interface{} "Unauthorized"
// @Failure      403      {object}  map[string]interface{} "Forbidden"
// @Failure      404      {object}  map[string]interface{} "Not Found"
// @Failure      409      {object}  map[string]interface{} "Conflict"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /irrigation/commands [post]
func (server *Server) CreateIrrigationCommand(ctx *gin.Context) {
	var req dto.CreateIrrigationCommandRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	deviceUUID, err := uuid.Parse(req.DeviceID)
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

	if !device.UserID.Valid || device.UserID.Int64 != authPayload.UserID {
		err := errors.New("device doesn't belong to authenticated user")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	if !strings.EqualFold(strings.TrimSpace(device.Status), "ativo") {
		err := errors.New("device is inactive, cannot send command")
		ctx.JSON(http.StatusConflict, errorResponse(err))
		return
	}

	pending, err := server.store.ExistsPendingIrrigationCommand(ctx, device.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if pending {
		ctx.JSON(http.StatusConflict,
			errorResponse(errors.New("device already has a pending command")))
		return
	}

	active, err := server.store.ExistsActiveIrrigationAction(ctx, device.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	switch req.Action {
	case "START":
		if active {
			ctx.JSON(http.StatusConflict,
				errorResponse(errors.New("device is already irrigating")))
			return
		}

	case "STOP":
		if !active {
			ctx.JSON(http.StatusConflict,
				errorResponse(errors.New("device is not irrigating")))
			return
		}
	}

	var durationSeconds sql.NullInt32
	if req.Duration != nil {
		durationSeconds = sql.NullInt32{Int32: *req.Duration, Valid: true}
	}

	command, err := server.store.CreateIrrigationCommand(ctx, db.CreateIrrigationCommandParams{
		DeviceID:        device.ID,
		UserID:          authPayload.UserID,
		Action:          req.Action,
		DurationSeconds: durationSeconds,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	payload := dto.IrrigationCommandPayload{
		CommandID: command.Uuid.String(),
		Action:    command.Action,
	}
	if durationSeconds.Valid {
		payload.DurationSeconds = &durationSeconds.Int32
	}

	data, err := json.Marshal(payload)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if err := mqtt.PublishCommand(server.mqttClient, device.DeviceUid, data); err != nil {
		_, updateErr := server.store.UpdateIrrigationCommandStatus(context.Background(), db.UpdateIrrigationCommandStatusParams{
			Uuid:   command.Uuid,
			Status: "FAILED",
			ErrorMessage: sql.NullString{
				String: err.Error(),
				Valid:  true,
			},
		})
		if updateErr != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(updateErr))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewIrrigationCommandResponse(command))
}

func (server *Server) UpdateIrrigationCommand(ctx *gin.Context) {

}

func (server *Server) ListIrrigationHistory(ctx *gin.Context) {}

func (server *Server) GetIrrigationHistory(ctx *gin.Context) {}

func (server *Server) GetIrrigationStatus(ctx *gin.Context) {}
