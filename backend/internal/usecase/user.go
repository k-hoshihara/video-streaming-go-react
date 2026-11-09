// Package usecase はアプリケーションの業務ロジックを実装する。
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
)

// UserRepository はユーザーの永続化を担う。
type UserRepository interface {
	// Create はユーザーを保存する。メールアドレスが登録済みなら domain.ErrEmailAlreadyExists を返す。
	Create(ctx context.Context, user *domain.User) error
}

// UserUsecase はユーザーに関する操作を提供する。
type UserUsecase struct {
	repo UserRepository

	// テストで差し替えられるように関数として持つ
	now          func() time.Time
	newID        func() (uuid.UUID, error)
	hashPassword func(password []byte) ([]byte, error)
}

// NewUserUsecase は UserUsecase を生成する。
func NewUserUsecase(repo UserRepository) *UserUsecase {
	return &UserUsecase{
		repo:  repo,
		now:   time.Now,
		newID: uuid.NewV7,
		hashPassword: func(password []byte) ([]byte, error) {
			return bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		},
	}
}

// SignupInput はユーザー登録の入力。
type SignupInput struct {
	Name     string
	Email    string
	Password string
}

// Signup はユーザーを登録する。
// メールアドレスは大文字小文字を区別せずに一意とするため、小文字に揃えて保存する。
func (u *UserUsecase) Signup(ctx context.Context, in SignupInput) (*domain.User, error) {
	// 仕様書の maxLength は文字数なので、マルチバイト文字だと bcrypt の上限（バイト数）を超えうる
	if len(in.Password) > domain.MaxPasswordBytes {
		return nil, domain.ErrPasswordTooLong
	}

	hash, err := u.hashPassword([]byte(in.Password))
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id, err := u.newID()
	if err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}

	user := &domain.User{
		ID:           id,
		Name:         in.Name,
		Email:        strings.ToLower(in.Email),
		PasswordHash: hash,
		CreatedAt:    u.now().UTC(),
	}
	if err := u.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}
