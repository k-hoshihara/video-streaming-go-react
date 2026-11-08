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
	now  func() time.Time
}

// NewUserUsecase は UserUsecase を生成する。
func NewUserUsecase(repo UserRepository) *UserUsecase {
	return &UserUsecase{repo: repo, now: time.Now}
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
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}

	user := &domain.User{
		ID:           id.String(),
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
