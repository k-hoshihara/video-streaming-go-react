package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
)

type fakeUserRepository struct {
	created *domain.User
	err     error
}

func (f *fakeUserRepository) Create(_ context.Context, user *domain.User) error {
	if f.err != nil {
		return f.err
	}
	f.created = user
	return nil
}

func TestUserUsecase_Signup(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	repo := &fakeUserRepository{}
	u := NewUserUsecase(repo)
	u.now = func() time.Time { return now }

	got, err := u.Signup(t.Context(), SignupInput{Name: "taro", Email: "Taro@Example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Signup() error = %v", err)
	}

	if got.ID == "" {
		t.Error("ID is empty")
	}
	if got.Name != "taro" {
		t.Errorf("Name = %q, want %q", got.Name, "taro")
	}
	if got.Email != "taro@example.com" {
		t.Errorf("Email = %q, want lowercased %q", got.Email, "taro@example.com")
	}
	if !got.CreatedAt.Equal(now) || got.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt = %v, want %v in UTC", got.CreatedAt, now)
	}
	if err := bcrypt.CompareHashAndPassword(got.PasswordHash, []byte("password123")); err != nil {
		t.Errorf("PasswordHash does not match the password: %v", err)
	}
	if repo.created != got {
		t.Error("the returned user was not passed to the repository")
	}
}

func TestUserUsecase_Signup_RepositoryError(t *testing.T) {
	t.Parallel()

	u := NewUserUsecase(&fakeUserRepository{err: domain.ErrEmailAlreadyExists})

	_, err := u.Signup(t.Context(), SignupInput{Name: "taro", Email: "taro@example.com", Password: "password123"})
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("Signup() error = %v, want %v", err, domain.ErrEmailAlreadyExists)
	}
}
