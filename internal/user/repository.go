package user

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*User, error)
}

type CreateUserInput struct {
	Name         string
	Email        string
	PasswordHash *string
	AuthProvider string
	GoogleID     *string
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	user := User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: input.PasswordHash,
		AuthProvider: input.AuthProvider,
		GoogleID:     input.GoogleID,
	}

	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *GormRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User

	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *GormRepository) FindByGoogleID(ctx context.Context, googleID string) (*User, error) {
	var user User

	err := r.db.WithContext(ctx).Where("google_id = ?", googleID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}
