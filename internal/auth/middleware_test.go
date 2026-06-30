package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cermin-backend/internal/user"

	"github.com/gin-gonic/gin"
)

func TestRequireAuthSetsCurrentUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := NewService(nil, "test-secret")
	token, err := service.createJWT(&user.User{
		ID:    42,
		Email: "budi@example.com",
	})
	if err != nil {
		t.Fatalf("create jwt: %v", err)
	}

	router := gin.New()
	router.GET("/me", RequireAuth(service), func(c *gin.Context) {
		userID, ok := CurrentUserID(c)
		if !ok {
			t.Fatal("expected user id in gin context")
		}

		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != `{"user_id":42}` {
		t.Fatalf("expected authenticated user id response, got %q", rec.Body.String())
	}
}

func TestRequireAuthRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/me", RequireAuth(NewService(nil, "test-secret")), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAuthRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/me", RequireAuth(NewService(nil, "test-secret")), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
