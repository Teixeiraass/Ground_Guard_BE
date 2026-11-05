package handler

import (
	"net/http"

	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
)

// GetSoilHealth godoc
// @Summary      Saúde do solo
// @Description  Avalia os sensores do usuário: estado atual de cada um (OK, DRY, WET, OFFLINE, FAULT, NO_DATA) contra a faixa ideal de umidade e o percentual do tempo dentro dessa faixa nas últimas 24h. A faixa ideal começa no limite de irrigação automática do sensor (padrão 35%) e vai até 80%.
// @Tags         statistics
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.SoilHealthResponse
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      500 {object} map[string]interface{} "Internal Server Error"
// @Router       /statistics/soil-health [get]
func (server *Server) GetSoilHealth(ctx *gin.Context) {
	authPayload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	health, err := server.SoilHealthService.GetSoilHealth(ctx, authPayload.UserID)
	if err != nil {
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, health)
}
