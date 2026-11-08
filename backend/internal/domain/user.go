// Package domain はアプリケーションのエンティティとドメインエラーを定義する。
package domain

import (
	"errors"
	"time"
)

// ErrEmailAlreadyExists はメールアドレスが登録済みのときに返す。
var ErrEmailAlreadyExists = errors.New("email already exists")

// User は登録済みのユーザー。
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash []byte
	CreatedAt    time.Time
}
