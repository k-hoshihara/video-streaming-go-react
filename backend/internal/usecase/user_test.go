package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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

var validInput = SignupInput{Name: "taro", Email: "Taro@Example.com", Password: "password123"}

func TestUserUsecase_Signup(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	repo := &fakeUserRepository{}
	u := NewUserUsecase(repo)
	u.now = func() time.Time { return now }

	got, err := u.Signup(t.Context(), validInput)
	if err != nil {
		t.Fatalf("Signup() error = %v", err)
	}

	if got.ID == uuid.Nil {
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

func TestUserUsecase_Signup_PasswordLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"72 bytes is allowed", strings.Repeat("a", 72), nil},
		{"73 bytes is rejected", strings.Repeat("a", 73), domain.ErrPasswordTooLong},
		// 30 文字だが 90 バイトなので拒否する
		{"multibyte over 72 bytes is rejected", strings.Repeat("あ", 30), domain.ErrPasswordTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := NewUserUsecase(&fakeUserRepository{})
			// テストを速くするためハッシュ計算は省く
			u.hashPassword = func(p []byte) ([]byte, error) { return p, nil }

			in := validInput
			in.Password = tt.password
			_, err := u.Signup(t.Context(), in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Signup() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserUsecase_Signup_Errors(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	tests := []struct {
		name  string
		setup func(u *UserUsecase)
		want  error
	}{
		{
			name:  "hash password fails",
			setup: func(u *UserUsecase) { u.hashPassword = func([]byte) ([]byte, error) { return nil, errBoom } },
			want:  errBoom,
		},
		{
			name:  "generate id fails",
			setup: func(u *UserUsecase) { u.newID = func() (uuid.UUID, error) { return uuid.Nil, errBoom } },
			want:  errBoom,
		},
		{
			name:  "email already exists",
			setup: func(u *UserUsecase) { u.repo = &fakeUserRepository{err: domain.ErrEmailAlreadyExists} },
			want:  domain.ErrEmailAlreadyExists,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := NewUserUsecase(&fakeUserRepository{})
			tt.setup(u)

			got, err := u.Signup(t.Context(), validInput)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Signup() error = %v, want %v", err, tt.want)
			}
			if got != nil {
				t.Errorf("Signup() user = %+v, want nil", got)
			}
		})
	}
}
