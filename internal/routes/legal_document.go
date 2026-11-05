package routes

import "github.com/gin-gonic/gin"

type LegalDocumentHandler interface {
	GetLegalDocument(c *gin.Context)
	ListLegalDocument(c *gin.Context)
	GetTermsAcceptanceStatus(c *gin.Context)
	AcceptLegalDocument(c *gin.Context)
}

func registerLegalDocumentRoutes(router gin.IRoutes, h LegalDocumentHandler) {
	router.GET("/legal-document/:uuid", h.GetLegalDocument)
	router.GET("/legal-documents", h.ListLegalDocument)
}

func registerAuthLegalDocumentRoutes(authRoutes gin.IRoutes, h LegalDocumentHandler) {
	authRoutes.GET("/legal-documents/terms/status", h.GetTermsAcceptanceStatus)
	authRoutes.POST("/legal-documents/:uuid/accept", h.AcceptLegalDocument)
}
