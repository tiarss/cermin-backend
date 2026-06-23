package router

import (
	"net/http"

	"cermin-backend/internal/docs"

	scalar "github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/gin-gonic/gin"
)

func registerDocsRoutes(r *gin.Engine) {
	r.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", docs.OpenAPIJSON)
	})

	r.GET("/docs", func(c *gin.Context) {
		htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
			SpecContent: string(docs.OpenAPIJSON),
			CustomOptions: scalar.CustomOptions{
				PageTitle: "Cermin Backend API",
			},
			DarkMode: true,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
	})
}
