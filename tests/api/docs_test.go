package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cermin-backend/internal/config"
	"cermin-backend/internal/router"

	"github.com/gin-gonic/gin"
)

func TestOpenAPIJSONRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode openapi json: %v", err)
	}

	if body["openapi"] != "3.0.3" {
		t.Fatalf("expected openapi version 3.0.3, got %v", body["openapi"])
	}
}

func TestScalarDocsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "@scalar/api-reference") {
		t.Fatal("expected Scalar API Reference script")
	}

	if !strings.Contains(body, "Cermin Backend API") {
		t.Fatal("expected Scalar page to include the API title")
	}
}
