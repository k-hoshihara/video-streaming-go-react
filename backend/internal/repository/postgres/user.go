// Package postgres は PostgreSQL を使う repository の実装。
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
)

// uniqueViolation は一意制約違反の SQLSTATE。
const uniqueViolation = "23505"

// UserRepository はユーザーを PostgreSQL に保存する。
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository は UserRepository を生成する。
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create はユーザーを保存する。メールアドレスが登録済みなら domain.ErrEmailAlreadyExists を返す。
// 重複は email の UNIQUE 制約で検出するため、同時登録でも重複しない。
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, name, email, password_hash, created_at) VALUES ($1, $2, $3, $4, $5)`,
		user.ID, user.Name, user.Email, string(user.PasswordHash), user.CreatedAt,
	)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
		return domain.ErrEmailAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// FindByEmail はメールアドレスでユーザーを探す。見つからなければ domain.ErrUserNotFound を返す。
// メールアドレスは保存時に小文字にそろえているため、呼び出し側でも小文字にして渡す。
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var (
		user domain.User
		hash string
	)
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash, created_at FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Name, &user.Email, &hash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select user: %w", err)
	}
	user.PasswordHash = []byte(hash)
	return &user, nil
}
