package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	appoauth "github.com/Teixeiraass/ground_guard_be/internal/oauth"
	"github.com/gin-gonic/gin"
)

// OAuthLogin
// @Summary      Login via OAuth 2.0
// @Description  Autentica um usuário com Google ou Apple via id_token e cria a sessão local.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        provider      path      string                     true  "google ou apple"
// @Param        request       body      dto.OAuthLoginRequest       true  "ID token do provider"
// @Success      200           {object}  dto.LoginUserResponse
// @Failure      400           {object}  map[string]interface{} "Bad Request"
// @Failure      401           {object}  map[string]interface{} "Unauthorized"
// @Failure      500           {object}  map[string]interface{} "Internal Server Error"
// @Router       /users/oauth/{provider} [post]
func (server *Server) OAuthLogin(ctx *gin.Context) {
	var req dto.OAuthLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	provider := strings.ToLower(strings.TrimSpace(ctx.Param("provider")))
	if provider != appoauth.ProviderGoogle && provider != appoauth.ProviderApple {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("unsupported oauth provider")))
		return
	}

	identity, err := server.oauthService.Authenticate(ctx, provider, req.IDToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	user, err := server.createOAuthUser(ctx, oauthIdentity{
		Provider:      identity.Provider,
		Subject:       identity.Subject,
		Email:         identity.Email,
		FullName:      identity.FullName,
		Username:      identity.Username,
		EmailVerified: identity.EmailVerified,
		Picture:       identity.Picture,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp, err := server.createLoginResponse(ctx, user, ctx.Request.UserAgent(), ctx.ClientIP())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, rsp)
}
