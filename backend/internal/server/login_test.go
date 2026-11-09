package server_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/server"
)

// signup はテストの前提となるユーザーを登録し、ユーザー ID を返す。
func signup(t *testing.T, srv http.Handler, email, password string) string {
	t.Helper()

	code, body := doJSON(t, srv, http.MethodPost, "/v1/signup",
		`{"name":"taro","email":"`+email+`","password":"`+password+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("signup status = %d, body = %v", code, body)
	}
	return body["id"].(string)
}

func loginBody(email, password string) string {
	return `{"email":"` + email + `","password":"` + password + `"}`
}

func TestLogin_OK(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	userID := signup(t, srv, "taro@example.com", "password123")

	code, body := doJSON(t, srv, http.MethodPost, "/v1/login", loginBody("taro@example.com", "password123"))
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %v", code, http.StatusOK, body)
	}
	if body["token_type"] != "Bearer" || body["expires_in"] != float64(3600) {
		t.Errorf("body = %v, want token_type=Bearer expires_in=3600", body)
	}

	// 発行されたトークンは、ログインしたユーザーの ID を sub に持ち、1 時間後に期限切れになる
	token, _ := body["access_token"].(string)
	var claims jwt.RegisteredClaims
	if _, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) { return testJWTSecret, nil },
		jwt.WithValidMethods([]string{"HS256"})); err != nil {
		t.Fatalf("parse access_token: %v", err)
	}
	if claims.Subject != userID {
		t.Errorf("sub = %q, want %q", claims.Subject, userID)
	}
	if ttl := claims.ExpiresAt.Sub(claims.IssuedAt.Time); ttl != time.Hour {
		t.Errorf("exp - iat = %v, want %v", ttl, time.Hour)
	}
}

// メールアドレスは大文字小文字を区別しない。
func TestLogin_CaseInsensitiveEmail(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	signup(t, srv, "Taro@Example.com", "password123")

	for _, email := range []string{"taro@example.com", "TARO@EXAMPLE.COM"} {
		if code, body := doJSON(t, srv, http.MethodPost, "/v1/login", loginBody(email, "password123")); code != http.StatusOK {
			t.Errorf("login with %q: status = %d, body = %v", email, code, body)
		}
	}
}

// メールアドレスが登録されていない場合とパスワードが違う場合は、同じレスポンスを返す。
func TestLogin_Unauthorized(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	signup(t, srv, "taro@example.com", "password123")

	tests := []struct {
		name string
		body string
	}{
		{"wrong password", loginBody("taro@example.com", "password124")},
		{"unknown email", loginBody("jiro@example.com", "password123")},
		{"password of another case", loginBody("taro@example.com", "PASSWORD123")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, body := doJSON(t, srv, http.MethodPost, "/v1/login", tt.body)
			if code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d, body = %v", code, http.StatusUnauthorized, body)
			}
			if body["message"] != "invalid email or password" {
				t.Errorf("message = %v, want %q", body["message"], "invalid email or password")
			}
		})
	}
}

// 仕様書の minLength / maxLength と bcrypt の上限（72 バイト）について、境界の両側を確認する。
func TestLogin_Boundaries(t *testing.T) {
	t.Parallel()

	const (
		email    = "taro@example.com"
		password = "password123"
	)
	tests := []struct {
		name     string
		password string // 登録するパスワード。空なら password で登録する
		body     string
		want     int
	}{
		// password: 1〜72 文字
		{"password 0 chars", "", loginBody(email, ""), http.StatusBadRequest},
		{"password 1 char", "", loginBody(email, "a"), http.StatusUnauthorized},
		{"password 72 chars", strings.Repeat("a", 72), loginBody(email, strings.Repeat("a", 72)), http.StatusOK},
		{"password 73 chars", "", loginBody(email, strings.Repeat("a", 73)), http.StatusBadRequest},
		// password: 72 バイトまで。仕様書の maxLength（文字数）は満たすが、72 バイトを超えると一致するユーザーはいない
		{"password 72 bytes multibyte", strings.Repeat("あ", 24), loginBody(email, strings.Repeat("あ", 24)), http.StatusOK},
		{"password 73 bytes multibyte", "", loginBody(email, strings.Repeat("あ", 24)+"a"), http.StatusUnauthorized},
		// email: 254 文字まで（登録されていないため 401）
		{"email 254 chars", "", loginBody(email254, password), http.StatusUnauthorized},
		{"email 255 chars", "", loginBody(email254[:64]+"@"+labelOf(63)+"."+labelOf(63)+"."+labelOf(62), password), http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newTestServer(t)
			registered := tt.password
			if registered == "" {
				registered = password
			}
			signup(t, srv, email, registered)

			code, res := doJSON(t, srv, http.MethodPost, "/v1/login", tt.body)
			if code != tt.want {
				t.Fatalf("status = %d, want %d, body = %v", code, tt.want, res)
			}
		})
	}
}

func TestLogin_BadRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{"missing email", `{"password":"password123"}`},
		{"missing password", `{"email":"taro@example.com"}`},
		{"invalid email", loginBody("not-an-email", "password123")},
		{"email with NUL", loginBody(`ta\u0000ro@example.com`, "password123")},
		{"unknown key", `{"email":"taro@example.com","password":"password123","name":"taro"}`},
		{"case-variant email key", `{"email":"taro@example.com","password":"password123","EMAIL":"jiro@example.com"}`},
		{"malformed json", `{"email":`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newTestServer(t)

			code, body := doJSON(t, srv, http.MethodPost, "/v1/login", tt.body)
			if code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body = %v", code, http.StatusBadRequest, body)
			}
			if body["message"] == nil {
				t.Errorf("body = %v, want message", body)
			}
		})
	}
}

// 想定外のエラーは 500 にし、エラーの詳細をクライアントに返さない。
func TestLogin_InternalServerError(t *testing.T) {
	t.Parallel()

	e, err := server.New(failingHandler{}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("server.New() error = %v", err)
	}

	code, body := doJSON(t, e, http.MethodPost, "/v1/login", loginBody("taro@example.com", "password123"))
	if code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body = %v", code, http.StatusInternalServerError, body)
	}
	if body["message"] != "Internal Server Error" {
		t.Errorf("message = %v, want %q", body["message"], "Internal Server Error")
	}
}
