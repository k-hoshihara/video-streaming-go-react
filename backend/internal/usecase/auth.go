package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
)

// dummyPasswordHash は、メールアドレスが登録されていないときにパスワードの照合に使うハッシュ。
// 登録済みのときと同じ重さの bcrypt の照合をして、処理時間の差から登録の有無を推測されないようにする。
// コストは本物のハッシュ（bcrypt.DefaultCost）と同じ 10 にしている。
const dummyPasswordHash = "$2a$10$PgKLdBD5CQ6e56h.A9VEdurIjHxhoodJ0PHybp20Grh0Kie/.duBG" //nolint:gosec // 誰のパスワードでもない、照合時間をそろえるためのハッシュ

// UserFinder はユーザーを検索する。
type UserFinder interface {
	// FindByEmail はメールアドレスでユーザーを探す。見つからなければ domain.ErrUserNotFound を返す。
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

// TokenIssuer はアクセストークンを発行する。
type TokenIssuer interface {
	Issue(userID uuid.UUID) (string, error)
	TTL() time.Duration
}

// AuthUsecase は認証に関する操作を提供する。
type AuthUsecase struct {
	users  UserFinder
	tokens TokenIssuer

	// テストで差し替えられるように関数として持つ
	comparePassword func(hash, password []byte) error
}

// NewAuthUsecase は AuthUsecase を生成する。
func NewAuthUsecase(users UserFinder, tokens TokenIssuer) *AuthUsecase {
	return &AuthUsecase{
		users:           users,
		tokens:          tokens,
		comparePassword: bcrypt.CompareHashAndPassword,
	}
}

// LoginInput はログインの入力。
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput はログインの結果。
type LoginOutput struct {
	AccessToken string
	ExpiresIn   time.Duration
}

// Login はメールアドレスとパスワードを確かめ、アクセストークンを発行する。
// メールアドレスが登録されていない場合とパスワードが違う場合は、どちらも domain.ErrInvalidCredentials を返す。
func (u *AuthUsecase) Login(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	// 登録時に 72 バイトを超えるパスワードは受け付けていないため、一致するユーザーはいない
	if len(in.Password) > domain.MaxPasswordBytes {
		return nil, domain.ErrInvalidCredentials
	}

	// 登録時と同じく、メールアドレスは小文字にそろえて探す
	user, err := u.users.FindByEmail(ctx, strings.ToLower(in.Email))
	if errors.Is(err, domain.ErrUserNotFound) {
		_ = u.comparePassword([]byte(dummyPasswordHash), []byte(in.Password))
		return nil, domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	err = u.comparePassword(user.PasswordHash, []byte(in.Password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return nil, domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("compare password: %w", err)
	}

	token, err := u.tokens.Issue(user.ID)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}
	return &LoginOutput{AccessToken: token, ExpiresIn: u.tokens.TTL()}, nil
}
