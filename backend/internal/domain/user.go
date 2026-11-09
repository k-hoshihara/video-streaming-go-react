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
	// ErrUserNotFound はユーザーが見つからないときに返す。
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidCredentials はログインでメールアドレスまたはパスワードが違うときに返す。
	// 登録の有無を知られないよう、どちらが違うかは区別しない。
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// User は登録済みのユーザー。
type User struct {
	ID   uuid.UUID
	Name string

	// Email はメールアドレス。小文字にそろえて保存し、大文字小文字を区別せずに一意とする。
	//
	// 形式は仕様書（docs/api/v1/openapi.yaml）の pattern と format: email で検証する。
	// ブラウザの <input type="email">（WHATWG）と同じ形式に、次の 3 点を加えたもの。
	//   - @ の前は 64 文字まで（RFC 5321）
	//   - ドメインに . が 1 つ以上必要
	//   - @ の前の . は先頭・末尾・連続を不可（RFC 5322。format: email の生成コードが検証する）
	//
	// 全体は 254 文字まで。ASCII のみで、日本語や絵文字は使えない。
	//
	// 想定するパターン（OK）:
	//   - taro@example.com
	//   - Taro@Example.com              大文字を含む（taro@example.com として保存する）
	//   - taro+test@example.com         + や . などの記号
	//   - taro.yamada@mail.example.co.jp サブドメイン
	//   - taro@my-example.com           ドメインの途中の -
	//   - 1234@example.com              数字のみ
	//
	// 想定するパターン（NG）:
	//   - taro@localhost                ドメインに . がない
	//   - 山田@example.jp、taro@例え.jp  ASCII 以外
	//   - 😀@example.com                絵文字
	//   - ta ro@example.com             空白
	//   - "taro"@example.com            引用符（RFC では正しいが、扱えないサービスが多い）
	//   - taro@[192.168.0.1]            IP アドレス
	//   - taro@@example.com、taro      @ が 2 つ・ない
	//   - @example.com、taro@          @ の前後が空
	//   - taro@-example.com、taro@example-.com
	//                                   ドメインのラベルの先頭・末尾の -
	//   - taro@example..com            ドメインの . が連続
	//   - .taro@example.com、taro.@example.com、ta..ro@example.com
	//                                   @ の前の . が先頭・末尾・連続
	//
	// 境界値:
	//   - @ の前: 64 文字は OK、65 文字は NG
	//   - ドメインのラベル（. で区切った各部分）: 63 文字は OK、64 文字は NG
	//   - 全体: 254 文字は OK、255 文字は NG
	Email string

	PasswordHash []byte
	CreatedAt    time.Time
}
