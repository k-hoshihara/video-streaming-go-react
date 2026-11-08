// Package domain はアプリケーションのエンティティとドメインエラーを定義する。
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MaxPasswordBytes はパスワードの最大バイト数。
// bcrypt は 72 バイトまでしか扱えないため、文字数ではなくバイト数で制限する。
const MaxPasswordBytes = 72

var (
	// ErrEmailAlreadyExists はメールアドレスが登録済みのときに返す。
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrPasswordTooLong はパスワードが MaxPasswordBytes を超えるときに返す。
	ErrPasswordTooLong = errors.New("password is too long")
)

// User は登録済みのユーザー。
type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash []byte
	CreatedAt    time.Time
}
