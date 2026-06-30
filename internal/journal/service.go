package journal

import (
	"context"
	"errors"
	"strings"

	"cermin-backend/internal/user"
)

var (
	ErrJournalEntryNotFound = errors.New("journal entry not found")
	ErrEmptyJournalContent  = errors.New("journal content is required")
)

type ReflectionGenerator interface {
	GenerateReflection(ctx context.Context, input ReflectionInput) (*ReflectionData, error)
}

type ReflectionInput struct {
	Content     string
	UserName    *string
	ContextData *string
}

type ReflectionData struct {
	Validation      string         `json:"validation"`
	Summary         []string       `json:"summary"`
	GrowthPrompt    string         `json:"growthPrompt"`
	Emotions        []EmotionScore `json:"emotions"`
	DominantEmotion string         `json:"dominantEmotion"`
	HiddenLanguage  []string       `json:"hiddenLanguage,omitempty"`
}

type EmotionScore struct {
	Label      string  `json:"label"`
	Percentage float64 `json:"percentage"`
}

type CreateJournalInput struct {
	UserID      int64
	Title       *string
	Content     string
	IsCrisis    bool
	ContextData *string
}

type Service struct {
	journals  Repository
	users     user.Repository
	generator ReflectionGenerator
}

func NewService(journals Repository, users user.Repository, generator ReflectionGenerator) *Service {
	return &Service{
		journals:  journals,
		users:     users,
		generator: generator,
	}
}

func (s *Service) Create(ctx context.Context, input CreateJournalInput) (*JournalEntry, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrEmptyJournalContent
	}

	var userName *string
	if s.users != nil {
		foundUser, err := s.users.FindByID(ctx, input.UserID)
		if err != nil {
			return nil, err
		}
		userName = &foundUser.Name
	}

	reflection, err := s.generator.GenerateReflection(ctx, ReflectionInput{
		Content:     content,
		UserName:    userName,
		ContextData: input.ContextData,
	})
	if err != nil {
		return nil, err
	}

	return s.journals.CreateWithReflection(ctx, CreateJournalEntryRequest{
		UserID:   input.UserID,
		Title:    input.Title,
		Content:  content,
		IsCrisis: input.IsCrisis,
	}, *reflection)
}
