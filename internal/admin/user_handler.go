package admin

import (
	"errors"
	"net/http"
	"strconv"

	"cermin-backend/internal/user"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *user.Service
}

func NewUserHandler(service *user.Service) *UserHandler {
	return &UserHandler{service: service}
}

type createUserRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Email    string `json:"email" binding:"required,email,max=150"`
	Password string `json:"password" binding:"required,min=8"`
}

type updateUserRequest struct {
	Name     *string `json:"name" binding:"omitempty,min=1,max=100"`
	Email    *string `json:"email" binding:"omitempty,email,max=150"`
	Password *string `json:"password" binding:"omitempty,min=8"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var request createUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Create(c.Request.Context(), user.CreateAdminUserInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if errors.Is(err, user.ErrEmailAlreadyUsed) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *UserHandler) List(c *gin.Context) {
	page, err := parsePositiveIntQuery(c, "page", 1)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	perPage, err := parsePositiveIntQuery(c, "per_page", 10)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.List(c.Request.Context(), user.ListAdminUsersInput{
		Page:    page,
		PerPage: perPage,
		Search:  c.Query("search"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	result, err := h.service.Get(c.Request.Context(), id)
	if errors.Is(err, user.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var request updateUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Update(c.Request.Context(), id, user.UpdateAdminUserInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if errors.Is(err, user.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, user.ErrEmailAlreadyUsed) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	err := h.service.Delete(c.Request.Context(), id)
	if errors.Is(err, user.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func parseIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return 0, false
	}

	return id, true
}

func parsePositiveIntQuery(c *gin.Context, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil || parsedValue < 1 {
		return 0, errors.New(key + " must be a positive integer")
	}

	return parsedValue, nil
}
