package user

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	List(ctx context.Context, input ListUsersInput) ([]User, int64, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*User, error)
	Update(ctx context.Context, id int64, input UpdateUserInput) (*User, error)
	Delete(ctx context.Context, id int64) error
}

type CreateUserInput struct {
	Name         string
	Email        string
	PasswordHash *string
	AuthProvider string
	GoogleID     *string
}

type ListUsersInput struct {
	Page    int
	PerPage int
	Search  string
}

type UpdateUserInput struct {
	Name         *string
	Email        *string
	PasswordHash *string
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

func (r *GormRepository) List(ctx context.Context, input ListUsersInput) ([]User, int64, error) {
	var users []User
	var total int64

	page := input.Page
	if page < 1 {
		page = 1
	}

	perPage := input.PerPage
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}

	query := r.db.WithContext(ctx).Model(&User{})
	search := strings.TrimSpace(input.Search)
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Limit(perPage).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *GormRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	var user User

	err := r.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
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

func (r *GormRepository) Update(ctx context.Context, id int64, input UpdateUserInput) (*User, error) {
	foundUser, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		foundUser.Name = *input.Name
	}
	if input.Email != nil {
		foundUser.Email = *input.Email
	}
	if input.PasswordHash != nil {
		foundUser.PasswordHash = input.PasswordHash
	}

	if err := r.db.WithContext(ctx).Save(foundUser).Error; err != nil {
		return nil, err
	}

	return foundUser, nil
}

func (r *GormRepository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
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
