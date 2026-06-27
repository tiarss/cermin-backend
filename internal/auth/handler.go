package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	google  GoogleOAuth
	apple   AppleOAuth
}

func NewHandler(service *Service, google GoogleOAuth, apple AppleOAuth) *Handler {
	return &Handler{
		service: service,
		google:  google,
		apple:   apple,
	}
}

type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Register(c.Request.Context(), RegisterInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if errors.Is(err, ErrEmailAlreadyUsed) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *Handler) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Login(c.Request.Context(), LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if errors.Is(err, ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GoogleRedirect(c *gin.Context) {
	if h.google.ClientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "google oauth is not configured"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, h.google.AuthURL())
}

func (h *Handler) GoogleCallback(c *gin.Context) {
	if c.Query("state") != h.google.State {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid google oauth state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing google oauth code"})
		return
	}

	accessToken, err := h.google.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	googleUser, err := h.google.UserInfo(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.LoginOrCreateGoogleUser(c.Request.Context(), *googleUser)
	if errors.Is(err, ErrEmailAlreadyUsed) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) AppleRedirect(c *gin.Context) {
	if !h.apple.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "apple oauth is not configured"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, h.apple.AuthURL())
}

func (h *Handler) AppleCallback(c *gin.Context) {
	if !h.apple.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "apple oauth is not configured"})
		return
	}

	if h.apple.State != "" && callbackValue(c, "state") != h.apple.State {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid apple oauth state"})
		return
	}

	code := callbackValue(c, "code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing apple oauth code"})
		return
	}

	tokenResponse, err := h.apple.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	appleUser, err := h.apple.UserInfo(c.Request.Context(), tokenResponse.IDToken, callbackValue(c, "user"))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.LoginOrCreateAppleUser(c.Request.Context(), *appleUser)
	if errors.Is(err, ErrEmailAlreadyUsed) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func callbackValue(c *gin.Context, key string) string {
	if value := c.PostForm(key); value != "" {
		return value
	}

	return c.Query(key)
}
