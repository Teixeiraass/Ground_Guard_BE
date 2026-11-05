package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/middleware"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func firmwareError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidFirmwareVersion),
		errors.Is(err, service.ErrInvalidFirmwareImage),
		errors.Is(err, service.ErrFirmwareTooLarge):
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
	case errors.Is(err, service.ErrFirmwareVersionNotNewer),
		errors.Is(err, service.ErrNoFirmwareUpdate),
		errors.Is(err, service.ErrControllerOffline),
		errors.Is(err, service.ErrControllerIrrigating),
		errors.Is(err, service.ErrOtaInProgress):
		ctx.JSON(http.StatusConflict, errorResponse(err))
	case errors.Is(err, service.ErrControllerNotOwned):
		ctx.JSON(http.StatusForbidden, errorResponse(err))
	case errors.Is(err, service.ErrFirmwareSigningDisabled):
		ctx.JSON(http.StatusServiceUnavailable, errorResponse(err))
	case errors.Is(err, service.ErrNoFirmwareRelease), errors.Is(err, sql.ErrNoRows):
		ctx.JSON(http.StatusNotFound, errorResponse(err))
	default:
		serverError(ctx, err)
	}
}

// GetFirmwareManifest godoc
// @Summary      Manifesto do firmware (controladora)
// @Description  Usado pela controladora antes de se atualizar: versão mais recente, tamanho, SHA-256 e caminho relativo do download.
// @Tags         firmware
// @Produce      json
// @Success      200 {object} dto.FirmwareManifestResponse
// @Failure      404 {object} map[string]interface{} "Nenhuma versão publicada"
// @Router       /firmware/manifest [get]
func (server *Server) GetFirmwareManifest(ctx *gin.Context) {
	release, err := server.FirmwareService.LatestRelease(ctx)
	if err != nil {
		firmwareError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewFirmwareManifestResponse(release))
}

// DownloadFirmware godoc
// @Summary      Baixar imagem do firmware (controladora)
// @Tags         firmware
// @Produce      octet-stream
// @Param        uuid path string true "UUID da versão" Format(uuid)
// @Success      200 {file} binary
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Router       /firmware/{uuid}/download [get]
func (server *Server) DownloadFirmware(ctx *gin.Context) {
	var uri dto.FirmwareUriRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	release, err := server.FirmwareService.GetRelease(ctx, uuid.MustParse(uri.UUID))
	if err != nil {
		firmwareError(ctx, err)
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("X-Firmware-Version", release.Version)
	ctx.Header("X-Firmware-Sha256", release.Sha256)
	ctx.File(release.FilePath)
}

// GetLatestFirmware godoc
// @Summary      Última versão do firmware
// @Tags         firmware
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.FirmwareReleaseResponse
// @Failure      404 {object} map[string]interface{} "Nenhuma versão publicada"
// @Router       /firmware/latest [get]
func (server *Server) GetLatestFirmware(ctx *gin.Context) {
	release, err := server.FirmwareService.LatestRelease(ctx)
	if err != nil {
		firmwareError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewFirmwareReleaseResponse(release))
}

// GetControllerFirmware godoc
// @Summary      Situação do firmware da controladora
// @Description  Versão atual e mais recente, se há atualização, preferência de atualização automática, progresso da última atualização e, se houver, por que a atualização está bloqueada (OFFLINE, IRRIGATING, IN_PROGRESS).
// @Tags         firmware
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da controladora" Format(uuid)
// @Success      200 {object} dto.ControllerFirmwareResponse
// @Failure      403 {object} map[string]interface{} "Forbidden"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Router       /controller-devices/{uuid}/firmware [get]
func (server *Server) GetControllerFirmware(ctx *gin.Context) {
	var uri dto.ControllerUriRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	rsp, err := server.FirmwareService.GetControllerFirmware(ctx, payload.UserID, uuid.MustParse(uri.UUID))
	if err != nil {
		firmwareError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, rsp)
}

// UpdateControllerFirmware godoc
// @Summary      Atualizar firmware da controladora agora
// @Description  Envia o comando de atualização. A controladora baixa a versão publicada, confere o SHA-256, grava e reinicia; o progresso chega via WebSocket (controller_ota) e notificação.
// @Tags         firmware
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da controladora" Format(uuid)
// @Success      202 {object} dto.ControllerFirmwareResponse
// @Failure      403 {object} map[string]interface{} "Forbidden"
// @Failure      404 {object} map[string]interface{} "Not Found"
// @Failure      409 {object} map[string]interface{} "Sem atualização, offline, irrigando ou atualização em andamento"
// @Router       /controller-devices/{uuid}/firmware/update [post]
func (server *Server) UpdateControllerFirmware(ctx *gin.Context) {
	var uri dto.ControllerUriRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	rsp, err := server.FirmwareService.RequestUpdate(ctx, payload.UserID, uuid.MustParse(uri.UUID))
	if err != nil {
		firmwareError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, rsp)
}

// SetControllerAutoUpdate godoc
// @Summary      Preferência de atualização automática
// @Description  Com a opção ligada, a controladora se atualiza sozinha entre 02h e 05h (horário de Brasília), nunca durante uma irrigação.
// @Tags         firmware
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                   true "UUID da controladora" Format(uuid)
// @Param        request body dto.SetAutoUpdateRequest true "Preferência"
// @Success      200 {object} dto.ControllerFirmwareResponse
// @Failure      400 {object} map[string]interface{} "Bad Request"
// @Failure      403 {object} map[string]interface{} "Forbidden"
// @Router       /controller-devices/{uuid}/firmware/auto-update [put]
func (server *Server) SetControllerAutoUpdate(ctx *gin.Context) {
	var uri dto.ControllerUriRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req dto.SetAutoUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	payload := ctx.MustGet(middleware.AuthorizationPayloadKey).(*token.Payload)

	rsp, err := server.FirmwareService.SetAutoUpdate(ctx, payload.UserID, uuid.MustParse(uri.UUID), *req.Enabled)
	if err != nil {
		firmwareError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, rsp)
}

// PublishFirmware godoc
// @Summary      Publicar nova versão do firmware (admin)
// @Description  Envia o firmware.bin da controladora. A versão precisa ser maior que a última publicada. Donos de controladoras desatualizadas sem atualização automática recebem uma notificação.
// @Tags         admin
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file          formData file   true  "firmware.bin gerado pelo PlatformIO"
// @Param        version       formData string true  "Versão, ex.: 1.1.0 (igual ao FIRMWARE_VERSION do config.h)"
// @Param        build         formData string true  "Build, ex.: 2"
// @Param        release_notes formData string false "O que mudou"
// @Success      201 {object} dto.FirmwareReleaseResponse
// @Failure      400 {object} map[string]interface{} "Arquivo ou versão inválidos"
// @Failure      403 {object} map[string]interface{} "Acesso de admin necessário"
// @Failure      409 {object} map[string]interface{} "Versão não é maior que a última"
// @Router       /admin/firmware [post]
func (server *Server) PublishFirmware(ctx *gin.Context) {
	var form dto.PublishFirmwareForm
	if err := ctx.ShouldBind(&form); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	defer file.Close()

	release, err := server.FirmwareService.PublishRelease(ctx, form, file)
	if err != nil {
		firmwareError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, dto.NewFirmwareReleaseResponse(release))
}

// ListFirmwareReleases godoc
// @Summary      Listar versões do firmware (admin)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page_id   query int false "Página (default 1)"
// @Param        page_size query int false "Itens por página (default 20)"
// @Success      200 {array} dto.FirmwareReleaseResponse
// @Router       /admin/firmware [get]
func (server *Server) ListFirmwareReleases(ctx *gin.Context) {
	pageID, err := strconv.ParseInt(ctx.DefaultQuery("page_id", "1"), 10, 32)
	if err != nil || pageID < 1 {
		pageID = 1
	}
	pageSize, err := strconv.ParseInt(ctx.DefaultQuery("page_size", "20"), 10, 32)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	releases, err := server.FirmwareService.ListReleases(ctx, int32(pageSize), int32((pageID-1)*pageSize))
	if err != nil {
		firmwareError(ctx, err)
		return
	}
	rsp := make([]dto.FirmwareReleaseResponse, 0, len(releases))
	for _, r := range releases {
		rsp = append(rsp, dto.NewFirmwareReleaseResponse(r))
	}
	ctx.JSON(http.StatusOK, rsp)
}
