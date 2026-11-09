// Package auth はアクセストークン（JWT）を扱う。
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// MinSecretBytes は署名に使う秘密鍵の最小バイト数。
// HS256 の鍵は、ハッシュの出力と同じ 256 ビット（32 バイト）以上にする（RFC 7518 3.2）。
const MinSecretBytes = 32

// JWTIssuer はユーザーのアクセストークンを HS256 で署名して発行する。
type JWTIssuer struct {
	secret []byte
	ttl    time.Duration

	// テストで差し替えられるように関数として持つ
	now func() time.Time
}

// NewJWTIssuer は JWTIssuer を生成する。秘密鍵が MinSecretBytes より短ければエラーを返す。
func NewJWTIssuer(secret []byte, ttl time.Duration) (*JWTIssuer, error) {
	if len(secret) < MinSecretBytes {
		return nil, fmt.Errorf("jwt secret must be at least %d bytes, got %d", MinSecretBytes, len(secret))
	}
	return &JWTIssuer{secret: secret, ttl: ttl, now: time.Now}, nil
}

// Issue は userID を sub に入れたアクセストークンを発行する。
func (i *JWTIssuer) Issue(userID uuid.UUID) (string, error) {
	now := i.now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
	})
	return token.SignedString(i.secret)
}

// TTL はアクセストークンの有効期間を返す。
func (i *JWTIssuer) TTL() time.Duration {
	return i.ttl
}
