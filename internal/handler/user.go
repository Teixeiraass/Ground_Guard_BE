package handler

import (
	"database/sql"
	"io"
	"net/http"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

	hashPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewUserResponse(user))
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

	user, err := server.store.GetUser(ctx, payload.UserUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewUserResponse(user))
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
	}

	user, err := server.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, err := server.tokenMaker.CreateToken(
		user.Username,
		user.ID,
		user.Uuid,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, err := server.tokenMaker.CreateToken(
		user.Username,
		user.ID,
		user.Uuid,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := dto.LoginUserResponse{
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpiresAt: time.Now().Add(server.config.AccessTokenDuration),
		User:                 dto.NewUserResponse(user),
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

	payload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, err := server.tokenMaker.CreateToken(
		payload.Username,
		payload.UserID,
		payload.UserUUID,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.RenewAccessTokenResponse{AccessToken: accessToken})
}

// UpdateUserName
// @Summary      Atualiza o nome do usuário
// @Description  Atualiza o nome completo (full name) de um usuário específico utilizando o seu UUID.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        uuid    path      string                     true  "UUID do Usuário" Format(uuid)
// @Param        request body      dto.UpdateUserNameRequest  true  "Dados para atualização do nome"
// @Success      200     {object}  dto.UserResponse           "Usuário atualizado com sucesso"
// @Failure      400     {object}  object                     "Requisição inválida (Payload JSON ou UUID incorreto)"
// @Failure      500     {object}  object                     "Erro interno no servidor"
// @Router       /users/name/{uuid} [put]
func (server *Server) UpdateUserName(ctx *gin.Context) {
	var req dto.UpdateUserNameRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	uuidStr := ctx.Param("uuid")

	userUUID, err := uuid.Parse(uuidStr)
	if err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

	arg := db.UpdateUserNameParams{
		Uuid: userUUID,
		FullName: req.FullName,
	}

	user, err := server.store.UpdateUserName(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewUserResponse(user))
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
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }
    defer f.Close()

    imageData, err := io.ReadAll(f)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }

    imagePath, err := util.SaveUserImage(imageData, payload.Username)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }

    arg := db.UpdateUserProfileImageParams{
        Uuid: payload.UserUUID,
        ProfileImage: sql.NullString{
            String: imagePath,
            Valid:  true,
        },
    }

    user, err := server.store.UpdateUserProfileImage(ctx, arg)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }

    ctx.JSON(http.StatusOK, dto.NewUserResponse(user))
} 