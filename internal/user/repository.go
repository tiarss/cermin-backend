package user

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")

const postgresUniqueViolationCode = "23505"

type Repository interface {
	Create(ctx context.Context, request CreateUserRequest) (*User, error)
	List(ctx context.Context, request ListUsersRequest) ([]User, int64, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*User, error)
	FindByAppleID(ctx context.Context, appleID string) (*User, error)
	Update(ctx context.Context, id int64, request UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, id int64) error
}

type CreateUserRequest struct {
	Name         string
	Email        string
	PasswordHash *string
	AuthProvider string
	GoogleID     *string
	AppleID      *string
}

type ListUsersRequest struct {
	Page    int
	PerPage int
	Search  string
}

type UpdateUserRequest struct {
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

func (r *GormRepository) Create(ctx context.Context, request CreateUserRequest) (*User, error) {
	user := User{
		Name:         request.Name,
		Email:        request.Email,
		PasswordHash: request.PasswordHash,
		AuthProvider: request.AuthProvider,
		GoogleID:     request.GoogleID,
		AppleID:      request.AppleID,
	}

	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyUsed
		}
		return nil, err
	}

	return &user, nil
}

func (r *GormRepository) List(ctx context.Context, request ListUsersRequest) ([]User, int64, error) {
	var users []User
	var total int64

	query := r.db.WithContext(ctx).Model(&User{})
	search := strings.TrimSpace(request.Search)
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (request.Page - 1) * request.PerPage
	if err := query.Order("created_at DESC").Limit(request.PerPage).Offset(offset).Find(&users).Error; err != nil {
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

func (r *GormRepository) Update(ctx context.Context, id int64, request UpdateUserRequest) (*User, error) {
	foundUser, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if request.Name != nil {
		foundUser.Name = *request.Name
	}
	if request.Email != nil {
		foundUser.Email = *request.Email
	}
	if request.PasswordHash != nil {
		foundUser.PasswordHash = request.PasswordHash
	}

	if err := r.db.WithContext(ctx).Save(foundUser).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyUsed
		}
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

func (r *GormRepository) FindByAppleID(ctx context.Context, appleID string) (*User, error) {
	var user User

	err := r.db.WithContext(ctx).Where("apple_id = ?", appleID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == postgresUniqueViolationCode
}
