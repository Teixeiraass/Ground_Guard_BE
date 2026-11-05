package handler

import (
	"net/http"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/gin-gonic/gin"
)

// RequestPasswordReset
// @Summary      Solicita redefinição de senha
// @Description  Envia um email com um link para redefinir a senha, se o email existir. Sempre responde 204, pra não revelar se o email está cadastrado.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      dto.RequestPasswordResetRequest  true  "Email e token do captcha"
// @Success      204     "Solicitação processada"
// @Failure      400     {object}  map[string]interface{} "Payload inválido ou captcha reprovado"
// @Failure      500     {object}  map[string]interface{} "Erro interno no servidor"
// @Router       /password-reset/request [post]
func (server *Server) RequestPasswordReset(ctx *gin.Context) {
	var req dto.RequestPasswordResetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Quietly drop excess requests per address (still 204): stops someone
	// flooding a victim's inbox without revealing anything.
	if ok, _ := server.resetAccountLimiter.Allow("reset|" + service.NormalizeEmail(req.Email)); !ok {
		ctx.Status(http.StatusNoContent)
		return
	}

	err := server.UserService.RequestPasswordReset(ctx, service.NormalizeEmail(req.Email), req.TurnstileToken, ctx.ClientIP())
	if err != nil {
		serverError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ConfirmPasswordReset
// @Summary      Confirma a redefinição de senha
// @Description  Define a nova senha a partir do token de redefinição recebido por email. Chamado pela página web de redefinição.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      dto.ConfirmPasswordResetRequest  true  "Token, nova senha e token do captcha"
// @Success      204     "Senha redefinida com sucesso"
// @Failure      400     {object}  map[string]interface{} "Token inválido/expirado, captcha reprovado ou payload inválido"
// @Failure      500     {object}  map[string]interface{} "Erro interno no servidor"
// @Router       /password-reset/confirm [post]
func (server *Server) ConfirmPasswordReset(ctx *gin.Context) {
	var req dto.ConfirmPasswordResetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.UserService.ConfirmPasswordReset(ctx, req.Token, req.NewPassword, req.TurnstileToken, ctx.ClientIP())
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "não foi possível redefinir a senha, o link pode ter expirado"})
		return
	}

	ctx.Status(http.StatusNoContent)
}
