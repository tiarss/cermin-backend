package router

import (
	"net/http"

	"cermin-backend/internal/auth"
	"cermin-backend/internal/config"
	"cermin-backend/internal/user"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg config.Config) *gin.Engine {
	r := gin.Default()

	registerDocsRoutes(r)

	userRepository := user.NewRepository(db)
	authService := auth.NewService(userRepository, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService, auth.GoogleOAuth{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		State:        cfg.GoogleOAuthState,
	})

	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})

		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/register", authHandler.Register)
			authRoutes.POST("/login", authHandler.Login)
			authRoutes.GET("/google", authHandler.GoogleRedirect)
			authRoutes.GET("/google/callback", authHandler.GoogleCallback)
		}
	}

	return r
}
