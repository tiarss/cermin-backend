package journal

import (
	"errors"
	"net/http"

	"cermin-backend/internal/auth"

	"github.com/gin-gonic/gin"
)

type journalRequest struct {
	Title       *string `json:"title"`
	Content     string  `json:"content" binding:"required"`
	IsCrisis    bool    `json:"is_crisis"`
	ContextData *string `json:"context_data"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	var request journalRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	entry, err := h.service.Create(c.Request.Context(), CreateJournalInput{
		UserID:      auth.MustCurrentUserID(c),
		Title:       request.Title,
		Content:     request.Content,
		IsCrisis:    request.IsCrisis,
		ContextData: request.ContextData,
	})
	if errors.Is(err, ErrEmptyJournalContent) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entry)
}
