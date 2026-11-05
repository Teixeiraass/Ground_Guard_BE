package handler

import (
	"database/sql"
	"errors"
	"io"
	"net/http"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/gin-gonic/gin"
)

// CreateUser
// @Summary      Criar um novo usuário
// @Description  Registra um novo usuário no aplicativo ground guard
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreateUserRequest  true  "Dados de criação do usuário"
// @Success      201      {object}  dto.UserResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request"
// @Failure      403      {object}  map[string]interface{} "Forbidden (e.g., e-mail ou username já existe)"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /users [post]
func (server *Server) CreateUser(ctx *gin.Context) {
	var req dto.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.UserService.CreateUser(ctx, req)
	if err != nil {
		if err.Error() == "unique_violation" {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewUserResponse(*user))
}

// GetUser
// @Summary      Obter perfil do usuário logado
// @Description  Retorna as informações do usuário autenticado a partir do token
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200      {object}  dto.UserResponse
// @Failure      404      {object}  map[string]interface{} "User Not Found"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /users/me [get]
func (server *Server) GetUser(ctx *gin.Context) {
	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	user, err := server.UserService.GetUser(ctx, payload.UserUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewUserResponse(*user))
}

// LoginUser
// @Summary      Login de usuário
// @Description  Autentica um usuário e retorna os tokens de acesso e refresh
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginUserRequest  true  "Credenciais de login (email e senha)"
// @Success      200      {object}  dto.LoginUserResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request"
// @Failure      401      {object}  map[string]interface{} "Unauthorized (Senha incorreta)"
// @Failure      404      {object}  map[string]interface{} "User Not Found (Email não registrado)"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /users/login [post]
func (server *Server) LoginUser(ctx *gin.Context) {
	var req dto.LoginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Per-account limit on failed attempts, on top of the per-IP one: stops
	// password guessing against one account spread over many IPs.
	accountKey := "login|" + service.NormalizeEmail(req.Email)
	if blocked, wait := server.loginAccountLimiter.Blocked(accountKey); blocked {
		middleware.TooManyRequests(ctx, wait)
		return
	}

	rsp, err := server.UserService.LoginUser(ctx, req, ctx.Request.UserAgent(), ctx.ClientIP())
	if err != nil {
		// Unknown email and wrong password answer the same, so the login
		// can't be used to discover which emails have an account.
		if err.Error() == "unauthorized" {
			server.loginAccountLimiter.Hit(accountKey)
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		if errors.Is(err, service.ErrUserBanned) {
			ctx.JSON(http.StatusForbidden, gin.H{
				"error": "sua conta foi suspensa",
				"code":  "USER_BANNED",
			})
			return
		}

		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, rsp)
}

// RenewAccessToken
// @Summary      Renovar token de acesso
// @Description  Gera um novo token de acesso usando um refresh token válido
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RenewAccessTokenRequest  true  "Refresh Token"
// @Success      200      {object}  dto.RenewAccessTokenResponse
// @Failure      400      {object}  map[string]interface{} "Bad Request"
// @Failure      401      {object}  map[string]interface{} "Unauthorized (Token inválido ou expirado)"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /tokens/refresh [post]
func (server *Server) RenewAccessToken(ctx *gin.Context) {
	var req dto.RenewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	rsp, err := server.UserService.RenewAccessToken(ctx, req)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		if err.Error() == "unauthorized" || err.Error() == "blocked session" || err.Error() == "incorrect session user" || err.Error() == "mismatched session token" || err.Error() == "expired session" {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, rsp)
}

// LogoutUser godoc
// @Summary      Realizar logout do usuário
// @Description  Invalida a sessão atual do usuário, garantindo que o token não possa mais ser usado.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.LogoutUserRequest true "Dados para o logout (Ex: Session ID)"
// @Success      200 {object} map[string]string "Logout realizado com sucesso"
// @Failure      400 {object} map[string]any "Requisição inválida (JSON mal formatado ou faltando campos)"
// @Failure      401 {object} map[string]any "Não autorizado (Token inválido/ausente ou Sessão inválida para este usuário)"
// @Failure      500 {object} map[string]any "Erro interno do servidor ao bloquear a sessão"
// @Router       /users/logout [post]
func (server *Server) LogoutUser(ctx *gin.Context) {
	var req dto.LogoutUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	err := server.UserService.LogoutUser(ctx, req.SessionID, payload.UserID)
	if err != nil {
		if err.Error() == "invalid session" {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "logout successful",
	})
}

// UpdateUserName
// @Summary      Atualiza o nome do usuário
// @Description  Atualiza o nome completo (full name) do usuário autenticado.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      dto.UpdateUserNameRequest  true  "Dados para atualização do nome"
// @Success      200     {object}  dto.UserResponse           "Usuário atualizado com sucesso"
// @Failure      400     {object}  object                     "Requisição inválida (Payload JSON incorreto)"
// @Failure      500     {object}  object                     "Erro interno no servidor"
// @Router       /users/name [put]
func (server *Server) UpdateUserName(ctx *gin.Context) {
	var req dto.UpdateUserNameRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	user, err := server.UserService.UpdateUserName(ctx, payload.UserUUID, req.FullName)
	if err != nil {
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewUserResponse(*user))
}

// ChangePassword
// @Summary      Altera a senha do usuário autenticado
// @Description  Troca a senha do usuário logado, exigindo a senha atual como confirmação.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      dto.ChangePasswordRequest  true  "Senha atual e nova senha"
// @Success      204     "Senha alterada com sucesso"
// @Failure      400     {object}  map[string]interface{} "Senha atual incorreta ou payload inválido"
// @Failure      500     {object}  map[string]interface{} "Erro interno no servidor"
// @Router       /users/password [put]
func (server *Server) ChangePassword(ctx *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	err := server.UserService.ChangePassword(ctx, payload.UserUUID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, service.ErrCurrentPasswordIncorrect) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "senha atual incorreta"})
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// DeleteAccount
// @Summary      Exclui a conta do usuário autenticado
// @Description  Exclui (soft delete) a conta do usuário logado, exigindo a senha como confirmação. Revoga as sessões ativas.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      dto.DeleteAccountRequest  true  "Senha para confirmar a exclusão"
// @Success      204     "Conta excluída com sucesso"
// @Failure      400     {object}  map[string]interface{} "Senha incorreta ou payload inválido"
// @Failure      500     {object}  map[string]interface{} "Erro interno no servidor"
// @Router       /users/me [delete]
func (server *Server) DeleteAccount(ctx *gin.Context) {
	var req dto.DeleteAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	err := server.UserService.DeleteAccount(ctx, payload.UserUUID, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrCurrentPasswordIncorrect) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "senha incorreta"})
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// VerifyEmail
// @Summary      Confirma o email do usuário
// @Description  Confirma o email a partir do token recebido por email. Endpoint público (chamado pela página web de confirmação).
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body      dto.VerifyEmailRequest  true  "Token de verificação"
// @Success      204     "Email confirmado"
// @Failure      400     {object}  map[string]interface{} "Token inválido, expirado ou payload inválido"
// @Failure      500     {object}  map[string]interface{} "Erro interno no servidor"
// @Router       /users/verify-email [post]
func (server *Server) VerifyEmail(ctx *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.UserService.VerifyEmail(ctx, req.Token)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrExpiredToken) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "token inválido ou expirado"})
			return
		}
		serverError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ResendVerificationEmail
// @Summary      Reenvia o email de verificação
// @Description  Gera um novo token e reenvia o email de confirmação para o usuário autenticado.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      204     "Email reenviado (ou já estava verificado)"
// @Failure      500     {object}  map[string]interface{} "Erro interno no servidor"
// @Router       /users/verify-email/resend [post]
func (server *Server) ResendVerificationEmail(ctx *gin.Context) {
	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	if err := server.UserService.ResendVerificationEmail(ctx, payload.UserUUID); err != nil {
		serverError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// UpdateProfileImage
// @Summary      Atualiza a imagem de perfil
// @Description  Faz o upload e salva uma nova imagem de perfil para o usuário autenticado via Token
// @Tags         users
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        image    formData  file      true  "Arquivo da imagem de perfil (ex: PNG, JPG)"
// @Success      200      {object}  dto.UserResponse "Usuário atualizado com sucesso"
// @Failure      400      {object}  map[string]interface{} "Bad Request (Arquivo não enviado ou inválido)"
// @Failure      401      {object}  map[string]interface{} "Unauthorized"
// @Failure      500      {object}  map[string]interface{} "Internal Server Error"
// @Router       /users/profile-image [put]
func (server *Server) UpdateProfileImage(ctx *gin.Context) {
	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	file, err := ctx.FormFile("image")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	f, err := file.Open()
	if err != nil {
		serverError(ctx, err)
		return
	}
	defer f.Close()

	imageData, err := io.ReadAll(f)
	if err != nil {
		serverError(ctx, err)
		return
	}

	user, err := server.UserService.UpdateProfileImage(ctx, payload.UserUUID, imageData, payload.Username)
	if errors.Is(err, util.ErrInvalidImage) {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if err != nil {
		serverError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.NewUserResponse(*user))
}
