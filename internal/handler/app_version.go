package handler

import (
	"net/http"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/gin-gonic/gin"
)

// GetAppVersion godoc
// @Summary      Versão mínima/mais recente do app
// @Description  Endpoint público usado pelo app pra saber se precisa forçar atualização (abaixo da mínima) ou só avisar (abaixo da mais recente).
// @Tags         helper
// @Produce      json
// @Success      200 {object} dto.AppVersionResponse
// @Router       /app-version [get]
func (server *Server) GetAppVersion(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, dto.AppVersionResponse{
		MinimumVersion:  server.config.MinimumAppVersion,
		LatestVersion:   server.config.LatestAppVersion,
		AndroidStoreUrl: server.config.AndroidStoreUrl,
		IosStoreUrl:     server.config.IosStoreUrl,
	})
}
