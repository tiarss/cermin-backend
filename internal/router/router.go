package router

import (
	"net/http"

	"cermin-backend/internal/admin"
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
	userService := user.NewService(userRepository)
	userHandler := admin.NewUserHandler(userService)
	authService := auth.NewService(userRepository, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService, auth.GoogleOAuth{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		State:        cfg.GoogleOAuthState,
	}, auth.AppleOAuth{
		ClientID:    cfg.AppleClientID,
		TeamID:      cfg.AppleTeamID,
		KeyID:       cfg.AppleKeyID,
		PrivateKey:  cfg.ApplePrivateKey,
		RedirectURL: cfg.AppleRedirectURL,
		State:       cfg.AppleOAuthState,
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
			authRoutes.GET("/apple", authHandler.AppleRedirect)
			authRoutes.GET("/apple/callback", authHandler.AppleCallback)
			authRoutes.POST("/apple/callback", authHandler.AppleCallback)
		}

		adminRoutes := v1.Group("/admin")
		{
			userRoutes := adminRoutes.Group("/users")
			{
				userRoutes.POST("", userHandler.Create)
				userRoutes.GET("", userHandler.List)
				userRoutes.GET("/:id", userHandler.Get)
				userRoutes.PATCH("/:id", userHandler.Update)
				userRoutes.DELETE("/:id", userHandler.Delete)
			}
		}
	}

	return r
}
