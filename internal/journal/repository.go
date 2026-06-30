package journal

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	CreateWithReflection(ctx context.Context, request CreateJournalEntryRequest, reflection ReflectionData) (*JournalEntry, error)

	GetJournalEntriesByUserID(ctx context.Context, userID int64) ([]JournalEntry, error)
	GetJournalEntryByID(ctx context.Context, userID int64, entryID int64) (*JournalEntry, error)
	// UpdateJournalEntry(entry *JournalEntry) error
	Delete(ctx context.Context, userID int64, entryID int64) error
}

type CreateJournalEntryRequest struct {
	UserID   int64
	Title    *string
	Content  string
	IsCrisis bool
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{
		db: db,
	}
}

func (r *GormRepository) CreateWithReflection(ctx context.Context, request CreateJournalEntryRequest, reflection ReflectionData) (*JournalEntry, error) {
	var entry JournalEntry

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		entry = JournalEntry{
			UserID:   request.UserID,
			Title:    request.Title,
			Content:  request.Content,
			IsCrisis: request.IsCrisis,
		}
		if err := tx.Create(&entry).Error; err != nil {
			return err
		}

		dominantEmotion := EmotionLabel(reflection.DominantEmotion)
		createdReflection := JournalReflection{
			JournalEntryID:  entry.ID,
			Validation:      stringPtr(reflection.Validation),
			GrowthPrompt:    stringPtr(reflection.GrowthPrompt),
			DominantEmotion: &dominantEmotion,
		}
		if err := tx.Create(&createdReflection).Error; err != nil {
			return err
		}

		for position, summary := range reflection.Summary {
			if err := tx.Create(&ReflectionSummary{
				ReflectionID: createdReflection.ID,
				Position:     position,
				SummaryText:  summary,
			}).Error; err != nil {
				return err
			}
		}

		for position, hiddenLanguage := range reflection.HiddenLanguage {
			if err := tx.Create(&ReflectionHiddenLanguage{
				ReflectionID:   createdReflection.ID,
				Position:       position,
				HiddenLanguage: hiddenLanguage,
			}).Error; err != nil {
				return err
			}
		}

		for _, emotion := range reflection.Emotions {
			if err := tx.Create(&ReflectionEmotionScore{
				ReflectionID: createdReflection.ID,
				Label:        EmotionLabel(emotion.Label),
				Percentage:   emotion.Percentage,
			}).Error; err != nil {
				return err
			}
		}

		return tx.
			Preload("Reflection.Summaries").
			Preload("Reflection.HiddenLanguages").
			Preload("Reflection.EmotionScores").
			First(&entry, entry.ID).Error
	})
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (r *GormRepository) GetJournalEntriesByUserID(ctx context.Context, userID int64) ([]JournalEntry, error) {
	var entries []JournalEntry

	err := r.db.WithContext(ctx).
		Preload("Reflection.Summaries").
		Preload("Reflection.HiddenLanguages").
		Preload("Reflection.EmotionScores").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&entries).Error
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *GormRepository) GetJournalEntryByID(ctx context.Context, userID int64, entryID int64) (*JournalEntry, error) {
	var entry JournalEntry

	err := r.db.WithContext(ctx).
		Preload("Reflection.Summaries").
		Preload("Reflection.HiddenLanguages").
		Preload("Reflection.EmotionScores").
		Where("user_id = ?", userID).
		First(&entry, entryID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrJournalEntryNotFound
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (r *GormRepository) Delete(ctx context.Context, userID int64, entryID int64) error {
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&JournalEntry{}, entryID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrJournalEntryNotFound
	}

	return nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
