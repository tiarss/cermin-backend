package user

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var ErrEmailAlreadyUsed = errors.New("email already used")

type Service struct {
	users Repository
}

func NewService(users Repository) *Service {
	return &Service{users: users}
}

type CreateAdminUserInput struct {
	Name     string
	Email    string
	Password string
}

type ListAdminUsersInput struct {
	Page    int
	PerPage int
	Search  string
}

type UpdateAdminUserInput struct {
	Name     *string
	Email    *string
	Password *string
}

type ListAdminUsersResult struct {
	Data    []AdminUser `json:"data"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
}

func (s *Service) Create(ctx context.Context, input CreateAdminUserInput) (*AdminUser, error) {
	existingUser, err := s.users.FindByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyUsed
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	createdUser, err := s.users.Create(ctx, CreateUserInput{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: &passwordHash,
		AuthProvider: "local",
	})
	if err != nil {
		return nil, err
	}

	adminUser := ToAdminUser(createdUser)
	return &adminUser, nil
}

func (s *Service) List(ctx context.Context, input ListAdminUsersInput) (*ListAdminUsersResult, error) {
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

	users, total, err := s.users.List(ctx, ListUsersInput{
		Page:    page,
		PerPage: perPage,
		Search:  input.Search,
	})
	if err != nil {
		return nil, err
	}

	data := make([]AdminUser, 0, len(users))
	for index := range users {
		data = append(data, ToAdminUser(&users[index]))
	}

	return &ListAdminUsersResult{
		Data:    data,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*AdminUser, error) {
	foundUser, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	adminUser := ToAdminUser(foundUser)
	return &adminUser, nil
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateAdminUserInput) (*AdminUser, error) {
	if input.Email != nil {
		existingUser, err := s.users.FindByEmail(ctx, *input.Email)
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
		if existingUser != nil && existingUser.ID != id {
			return nil, ErrEmailAlreadyUsed
		}
	}

	repositoryInput := UpdateUserInput{
		Name:  input.Name,
		Email: input.Email,
	}

	if input.Password != nil {
		passwordHash, err := hashPassword(*input.Password)
		if err != nil {
			return nil, err
		}
		repositoryInput.PasswordHash = &passwordHash
	}

	updatedUser, err := s.users.Update(ctx, id, repositoryInput)
	if err != nil {
		return nil, err
	}

	adminUser := ToAdminUser(updatedUser)
	return &adminUser, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.users.Delete(ctx, id)
}

func hashPassword(password string) (string, error) {
	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(passwordHashBytes), nil
}
