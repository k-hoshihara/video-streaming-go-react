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

type fakeUserFinder struct {
	user  *domain.User
	err   error
	email string // 渡されたメールアドレス
	calls int
}

func (f *fakeUserFinder) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	f.email = email
	f.calls++
	return f.user, f.err
}

type fakeTokenIssuer struct {
	err    error
	userID uuid.UUID
}

func (f *fakeTokenIssuer) Issue(userID uuid.UUID) (string, error) {
	f.userID = userID
	if f.err != nil {
		return "", f.err
	}
	return "token-" + userID.String(), nil
}

func (f *fakeTokenIssuer) TTL() time.Duration { return time.Hour }

// minCostHash はテストを速くするため、最小のコストで作ったハッシュを返す。
func minCostHash(t *testing.T, password string) []byte {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestAuthUsecase_Login(t *testing.T) {
	t.Parallel()

	user := &domain.User{ID: uuid.Must(uuid.NewV7()), Email: "taro@example.com", PasswordHash: minCostHash(t, "password123")}
	users := &fakeUserFinder{user: user}
	tokens := &fakeTokenIssuer{}
	u := NewAuthUsecase(users, tokens)

	// 大文字を含むメールアドレスでもログインできる
	got, err := u.Login(t.Context(), LoginInput{Email: "Taro@Example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if users.email != "taro@example.com" {
		t.Errorf("FindByEmail() email = %q, want lowercased %q", users.email, "taro@example.com")
	}
	if tokens.userID != user.ID {
		t.Errorf("Issue() userID = %v, want %v", tokens.userID, user.ID)
	}
	if got.AccessToken != "token-"+user.ID.String() || got.ExpiresIn != time.Hour {
		t.Errorf("Login() = %+v", got)
	}
}

func TestAuthUsecase_Login_InvalidCredentials(t *testing.T) {
	t.Parallel()

	user := &domain.User{ID: uuid.Must(uuid.NewV7()), Email: "taro@example.com", PasswordHash: minCostHash(t, "password123")}
	tests := []struct {
		name     string
		users    *fakeUserFinder
		password string
	}{
		{"wrong password", &fakeUserFinder{user: user}, "password124"},
		{"user not found", &fakeUserFinder{err: domain.ErrUserNotFound}, "password123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := NewAuthUsecase(tt.users, &fakeTokenIssuer{})

			_, err := u.Login(t.Context(), LoginInput{Email: "taro@example.com", Password: tt.password})
			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want %v", err, domain.ErrInvalidCredentials)
			}
		})
	}
}

// ユーザーが見つからなくても、処理時間をそろえるためダミーのハッシュとパスワードを照合する。
func TestAuthUsecase_Login_UserNotFoundComparesDummyHash(t *testing.T) {
	t.Parallel()

	u := NewAuthUsecase(&fakeUserFinder{err: domain.ErrUserNotFound}, &fakeTokenIssuer{})
	var compared []byte
	u.comparePassword = func(hash, password []byte) error {
		compared = hash
		return bcrypt.CompareHashAndPassword(hash, password)
	}

	if _, err := u.Login(t.Context(), LoginInput{Email: "taro@example.com", Password: "password123"}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want %v", err, domain.ErrInvalidCredentials)
	}
	if string(compared) != dummyPasswordHash {
		t.Fatalf("compared hash = %q, want the dummy hash", compared)
	}
	// 本物のハッシュと同じコストで照合する
	if cost, err := bcrypt.Cost([]byte(dummyPasswordHash)); err != nil || cost != bcrypt.DefaultCost {
		t.Errorf("dummy hash cost = %d (%v), want %d", cost, err, bcrypt.DefaultCost)
	}
}

func TestAuthUsecase_Login_PasswordLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"72 bytes", strings.Repeat("a", 72), nil},
		{"73 bytes", strings.Repeat("a", 73), domain.ErrInvalidCredentials},
		// 24 文字 × 3 バイト = 72 バイト
		{"multibyte 72 bytes", strings.Repeat("あ", 24), nil},
		{"multibyte 73 bytes", strings.Repeat("あ", 24) + "a", domain.ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			users := &fakeUserFinder{user: &domain.User{ID: uuid.Must(uuid.NewV7()), PasswordHash: minCostHash(t, tt.password[:min(len(tt.password), 72)])}}
			u := NewAuthUsecase(users, &fakeTokenIssuer{})

			_, err := u.Login(t.Context(), LoginInput{Email: "taro@example.com", Password: tt.password})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
			}
			// 72 バイトを超えるパスワードは、DB を見ずに拒否する
			if tt.wantErr != nil && users.calls != 0 {
				t.Errorf("FindByEmail() was called %d times, want 0", users.calls)
			}
		})
	}
}

func TestAuthUsecase_Login_Errors(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	user := &domain.User{ID: uuid.Must(uuid.NewV7()), PasswordHash: minCostHash(t, "password123")}
	tests := []struct {
		name  string
		setup func(u *AuthUsecase)
	}{
		{"find user fails", func(u *AuthUsecase) { u.users = &fakeUserFinder{err: errBoom} }},
		// 保存されているハッシュが壊れている場合など、照合そのものの失敗は認証エラーにしない
		{"compare password fails", func(u *AuthUsecase) { u.comparePassword = func([]byte, []byte) error { return errBoom } }},
		{"issue token fails", func(u *AuthUsecase) { u.tokens = &fakeTokenIssuer{err: errBoom} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := NewAuthUsecase(&fakeUserFinder{user: user}, &fakeTokenIssuer{})
			tt.setup(u)

			got, err := u.Login(t.Context(), LoginInput{Email: "taro@example.com", Password: "password123"})
			if !errors.Is(err, errBoom) {
				t.Fatalf("Login() error = %v, want %v", err, errBoom)
			}
			if errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, must not be %v", err, domain.ErrInvalidCredentials)
			}
			if got != nil {
				t.Errorf("Login() = %+v, want nil", got)
			}
		})
	}
}
