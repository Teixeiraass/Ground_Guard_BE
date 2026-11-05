package handler

import (
	"html/template"
	"net/http"

	webassets "github.com/Teixeiraass/ground_guard_be/web"
	"github.com/gin-gonic/gin"
)

// ResetPasswordPage
// @Summary      Página de redefinição de senha
// @Description  Página HTML aberta a partir do link enviado por email. O usuário define a nova senha ali (chama POST /password-reset/confirm).
// @Tags         auth
// @Produce      html
// @Param        token  query  string  true  "Token de redefinição de senha recebido por email"
// @Success      200
// @Router       /reset-password [get]
func (server *Server) ResetPasswordPage(ctx *gin.Context) {
	tmpl, err := template.New("reset_password").Parse(webassets.ResetPasswordPage)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "erro ao carregar a página")
		return
	}

	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.Status(http.StatusOK)
	_ = tmpl.Execute(ctx.Writer, gin.H{"SiteKey": server.config.TurnstileSiteKey})
}

// VerifyEmailPage
// @Summary      Página de confirmação de email
// @Description  Página HTML aberta a partir do link enviado por email. Confirma o email ao clicar (chama POST /users/verify-email).
// @Tags         auth
// @Produce      html
// @Param        token  query  string  true  "Token de verificação de email recebido por email"
// @Success      200
// @Router       /verify-email [get]
func (server *Server) VerifyEmailPage(ctx *gin.Context) {
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.Status(http.StatusOK)
	_, _ = ctx.Writer.WriteString(webassets.VerifyEmailPage)
}
