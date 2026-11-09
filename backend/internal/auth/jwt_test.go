package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var testSecret = []byte(strings.Repeat("s", MinSecretBytes))

func TestNewJWTIssuer_SecretLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		secret  []byte
		wantErr bool
	}{
		{"empty", nil, true},
		{"31 bytes", []byte(strings.Repeat("s", MinSecretBytes-1)), true},
		{"32 bytes", []byte(strings.Repeat("s", MinSecretBytes)), false},
		// 文字数ではなくバイト数で数える（"あ" は 3 バイト）
		{"10 multibyte chars (30 bytes)", []byte(strings.Repeat("あ", 10)), true},
		{"11 multibyte chars (33 bytes)", []byte(strings.Repeat("あ", 11)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewJWTIssuer(tt.secret, time.Hour)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewJWTIssuer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWTIssuer_Issue(t *testing.T) {
	t.Parallel()

	issuer, err := NewJWTIssuer(testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	issuer.now = func() time.Time { return now }
	userID := uuid.Must(uuid.NewV7())

	signed, err := issuer.Issue(userID)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	var claims jwt.RegisteredClaims
	_, err = jwt.ParseWithClaims(signed, &claims, func(*jwt.Token) (any, error) { return testSecret, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("sub = %q, want %q", claims.Subject, userID)
	}
	if !claims.IssuedAt.Equal(now) {
		t.Errorf("iat = %v, want %v", claims.IssuedAt, now)
	}
	if !claims.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Errorf("exp = %v, want %v", claims.ExpiresAt, now.Add(time.Hour))
	}
	if issuer.TTL() != time.Hour {
		t.Errorf("TTL() = %v, want %v", issuer.TTL(), time.Hour)
	}
}

// 別の秘密鍵では検証できない（改ざんしたトークンを受け付けない）。
func TestJWTIssuer_Issue_WrongSecret(t *testing.T) {
	t.Parallel()

	issuer, err := NewJWTIssuer(testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := issuer.Issue(uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatal(err)
	}

	other := []byte(strings.Repeat("x", MinSecretBytes))
	if _, err := jwt.Parse(signed, func(*jwt.Token) (any, error) { return other, nil }); err == nil {
		t.Fatal("parse with another secret succeeded, want an error")
	}
}
