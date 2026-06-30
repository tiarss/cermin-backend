package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestServiceCreateHashesPassword(t *testing.T) {
	repository := newFakeRepository()
	service := NewService(repository)

	result, err := service.Create(context.Background(), CreateAdminUserInput{
		Name:     "Budi",
		Email:    "budi@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected create user to succeed: %v", err)
	}

	if result.AuthProvider != "local" {
		t.Fatalf("expected auth provider local, got %q", result.AuthProvider)
	}

	createdUser := repository.users[result.ID]
	if createdUser.PasswordHash == nil {
		t.Fatal("expected password hash to be stored")
	}
	if *createdUser.PasswordHash == "password123" {
		t.Fatal("expected password hash not to store the raw password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*createdUser.PasswordHash), []byte("password123")); err != nil {
		t.Fatalf("expected password hash to match raw password: %v", err)
	}
}

func TestServiceCreateRejectsDuplicateEmail(t *testing.T) {
	repository := newFakeRepository()
	repository.users[1] = &User{ID: 1, Name: "Budi", Email: "budi@example.com"}
	service := NewService(repository)

	_, err := service.Create(context.Background(), CreateAdminUserInput{
		Name:     "Siti",
		Email:    "budi@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrEmailAlreadyUsed) {
		t.Fatalf("expected ErrEmailAlreadyUsed, got %v", err)
	}
}

func TestServiceUpdateRejectsDuplicateEmail(t *testing.T) {
	repository := newFakeRepository()
	repository.users[1] = &User{ID: 1, Name: "Budi", Email: "budi@example.com"}
	repository.users[2] = &User{ID: 2, Name: "Siti", Email: "siti@example.com"}
	service := NewService(repository)

	email := "siti@example.com"
	_, err := service.Update(context.Background(), 1, UpdateAdminUserInput{
		Email: &email,
	})
	if !errors.Is(err, ErrEmailAlreadyUsed) {
		t.Fatalf("expected ErrEmailAlreadyUsed, got %v", err)
	}
}

type fakeRepository struct {
	users  map[int64]*User
	nextID int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		users:  make(map[int64]*User),
		nextID: 1,
	}
}

func (r *fakeRepository) Create(ctx context.Context, input CreateUserRequest) (*User, error) {
	user := &User{
		ID:           r.nextID,
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: input.PasswordHash,
		AuthProvider: input.AuthProvider,
		GoogleID:     input.GoogleID,
		AppleID:      input.AppleID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	r.users[user.ID] = user
	r.nextID++

	return user, nil
}

func (r *fakeRepository) List(ctx context.Context, input ListUsersRequest) ([]User, int64, error) {
	users := make([]User, 0, len(r.users))
	for _, storedUser := range r.users {
		users = append(users, *storedUser)
	}

	return users, int64(len(users)), nil
}

func (r *fakeRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (r *fakeRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, ErrUserNotFound
}

func (r *fakeRepository) FindByGoogleID(ctx context.Context, googleID string) (*User, error) {
	for _, user := range r.users {
		if user.GoogleID != nil && *user.GoogleID == googleID {
			return user, nil
		}
	}

	return nil, ErrUserNotFound
}

func (r *fakeRepository) FindByAppleID(ctx context.Context, appleID string) (*User, error) {
	for _, user := range r.users {
		if user.AppleID != nil && *user.AppleID == appleID {
			return user, nil
		}
	}

	return nil, ErrUserNotFound
}

func (r *fakeRepository) Update(ctx context.Context, id int64, input UpdateUserRequest) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.PasswordHash != nil {
		user.PasswordHash = input.PasswordHash
	}
	user.UpdatedAt = time.Now()

	return user, nil
}

func (r *fakeRepository) Delete(ctx context.Context, id int64) error {
	if _, ok := r.users[id]; !ok {
		return ErrUserNotFound
	}

	delete(r.users, id)
	return nil
}
