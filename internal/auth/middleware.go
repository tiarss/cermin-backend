package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserIDKey = "auth_user_id"
	ContextEmailKey  = "auth_email"
)

func RequireAuth(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		claims, err := service.ParseJWT(token)
		if errors.Is(err, ErrExpiredToken) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextEmailKey, claims.Email)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) (int64, bool) {
	value, ok := c.Get(ContextUserIDKey)
	if !ok {
		return 0, false
	}

	userID, ok := value.(int64)
	return userID, ok
}

func MustCurrentUserID(c *gin.Context) int64 {
	userID, ok := CurrentUserID(c)
	if !ok {
		panic("auth user id is missing from gin context")
	}

	return userID
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	return parts[1], true
}
