package journal

import (
	"time"

	"cermin-backend/internal/user"
)

type EmotionLabel string

const (
	EmotionSenang  EmotionLabel = "Senang"
	EmotionSedih   EmotionLabel = "Sedih"
	EmotionMarah   EmotionLabel = "Marah"
	EmotionCemas   EmotionLabel = "Cemas"
	EmotionTenang  EmotionLabel = "Tenang"
	EmotionLelah   EmotionLabel = "Lelah"
	EmotionHarapan EmotionLabel = "Harapan"
	EmotionLainnya EmotionLabel = "Lainnya"
)

type JournalEntry struct {
	ID         int64              `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int64              `json:"user_id" gorm:"not null;index"`
	User       user.User          `json:"-" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Title      *string            `json:"title" gorm:"size:255"`
	Content    string             `json:"content" gorm:"type:text;not null"`
	IsCrisis   bool               `json:"is_crisis" gorm:"not null;default:false"`
	Reflection *JournalReflection `json:"reflection,omitempty" gorm:"foreignKey:JournalEntryID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt  time.Time          `json:"created_at" gorm:"not null"`
	UpdatedAt  time.Time          `json:"updated_at" gorm:"not null"`
}

type JournalReflection struct {
	ID              int64                      `json:"id" gorm:"primaryKey;autoIncrement"`
	JournalEntryID  int64                      `json:"journal_entry_id" gorm:"not null;unique"`
	Validation      *string                    `json:"validation" gorm:"type:text"`
	GrowthPrompt    *string                    `json:"growth_prompt" gorm:"type:text"`
	DominantEmotion *EmotionLabel              `json:"dominant_emotion" gorm:"size:50"`
	Summaries       []ReflectionSummary        `json:"summaries,omitempty" gorm:"foreignKey:ReflectionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	HiddenLanguages []ReflectionHiddenLanguage `json:"hidden_languages,omitempty" gorm:"foreignKey:ReflectionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	EmotionScores   []ReflectionEmotionScore   `json:"emotion_scores,omitempty" gorm:"foreignKey:ReflectionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt       time.Time                  `json:"created_at" gorm:"not null"`
	UpdatedAt       time.Time                  `json:"updated_at" gorm:"not null"`
}

type ReflectionSummary struct {
	ID           int64             `json:"id" gorm:"primaryKey;autoIncrement"`
	ReflectionID int64             `json:"reflection_id" gorm:"not null;index;uniqueIndex:idx_reflection_summaries_position"`
	Position     int               `json:"position" gorm:"not null;uniqueIndex:idx_reflection_summaries_position"`
	SummaryText  string            `json:"summary_text" gorm:"type:text;not null"`
	Reflection   JournalReflection `json:"-" gorm:"foreignKey:ReflectionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type ReflectionHiddenLanguage struct {
	ID             int64             `json:"id" gorm:"primaryKey;autoIncrement"`
	ReflectionID   int64             `json:"reflection_id" gorm:"not null;index;uniqueIndex:idx_reflection_hidden_languages_position"`
	Position       int               `json:"position" gorm:"not null;uniqueIndex:idx_reflection_hidden_languages_position"`
	HiddenLanguage string            `json:"hidden_language" gorm:"type:text;not null"`
	Reflection     JournalReflection `json:"-" gorm:"foreignKey:ReflectionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type ReflectionEmotionScore struct {
	ID           int64             `json:"id" gorm:"primaryKey;autoIncrement"`
	ReflectionID int64             `json:"reflection_id" gorm:"not null;index;uniqueIndex:idx_reflection_emotion_scores_label"`
	Label        EmotionLabel      `json:"label" gorm:"size:50;not null;uniqueIndex:idx_reflection_emotion_scores_label"`
	Percentage   float64           `json:"percentage" gorm:"type:numeric(5,2);not null"`
	Reflection   JournalReflection `json:"-" gorm:"foreignKey:ReflectionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Journal = JournalEntry
