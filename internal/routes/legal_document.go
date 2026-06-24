package routes

import "github.com/gin-gonic/gin"

type LegalDocumentHandler interface {
	GetLegalDocument(c *gin.Context)
	ListLegalDocument(c *gin.Context)
}

func registerLegalDocumentRoutes(router gin.IRoutes, h LegalDocumentHandler) {
	router.GET("/legal-document/:uuid", h.GetLegalDocument)
	router.GET("/legal-documents", h.ListLegalDocument)
}