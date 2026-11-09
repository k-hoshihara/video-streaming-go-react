package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/handler"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/oapi"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/repository/postgres"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/server"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/testdb"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/usecase"
)

// newTestServer は本番と同じ構成（仕様書による検証、PostgreSQL）で組み立てたサーバーを返す。
// DB はテストごとに専用のスキーマを使うため、並列に実行しても互いに影響しない。
func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	repo := postgres.NewUserRepository(testdb.New(t))
	h := handler.New(usecase.NewUserUsecase(repo))
	e, err := server.New(h, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("server.New() error = %v", err)
	}
	return e
}

func doJSON(t *testing.T, srv http.Handler, method, path, body string) (int, map[string]any) {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	raw, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("response is not a JSON object: %v, body = %s", err, raw)
	}
	return rec.Code, got
}

func TestSignup_Created(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)

	code, body := doJSON(t, srv, http.MethodPost, "/v1/signup",
		`{"name":"taro","email":"Taro@Example.com","password":"password123"}`)

	if code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %v", code, http.StatusCreated, body)
	}
	if body["name"] != "taro" || body["email"] != "taro@example.com" {
		t.Errorf("body = %v, want name=taro email=taro@example.com", body)
	}
	if body["id"] == nil || body["created_at"] == nil {
		t.Errorf("body = %v, want id and created_at", body)
	}
	if _, ok := body["password"]; ok {
		t.Error("response must not contain password")
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)

	if code, body := doJSON(t, srv, http.MethodPost, "/v1/signup",
		`{"name":"taro","email":"taro@example.com","password":"password123"}`); code != http.StatusCreated {
		t.Fatalf("first signup status = %d, body = %v", code, body)
	}

	// 大文字小文字だけが異なるメールアドレスも重複として扱う。
	code, body := doJSON(t, srv, http.MethodPost, "/v1/signup",
		`{"name":"jiro","email":"TARO@example.com","password":"password456"}`)
	if code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %v", code, http.StatusConflict, body)
	}
	if body["message"] == nil {
		t.Errorf("body = %v, want message", body)
	}
}

// labelOf は n 文字の a を返す。
func labelOf(n int) string { return strings.Repeat("a", n) }

// email254 は、@ の前が 64 文字、ドメインのラベルが 63 文字以下で、全体が 254 文字の正しいメールアドレス。
var email254 = labelOf(64) + "@" + labelOf(63) + "." + labelOf(63) + "." + labelOf(61)

// 仕様書の minLength / maxLength と bcrypt の上限（72 バイト）について、境界の両側を確認する。
func TestSignup_Boundaries(t *testing.T) {
	t.Parallel()

	body := func(name, email, password string) string {
		return `{"name":"` + name + `","email":"` + email + `","password":"` + password + `"}`
	}
	const (
		name     = "taro"
		email    = "taro@example.com"
		password = "password123"
	)

	tests := []struct {
		name string
		body string
		want int
	}{
		// name: 1〜50 文字
		{"name 0 chars", body("", email, password), http.StatusBadRequest},
		{"name 1 char", body("a", email, password), http.StatusCreated},
		{"name 50 chars", body(strings.Repeat("a", 50), email, password), http.StatusCreated},
		{"name 50 multibyte chars", body(strings.Repeat("あ", 50), email, password), http.StatusCreated},
		{"name 51 multibyte chars", body(strings.Repeat("あ", 51), email, password), http.StatusBadRequest},
		{"name with emoji", body("taro😀", email, password), http.StatusCreated},
		{"name 51 chars", body(strings.Repeat("a", 51), email, password), http.StatusBadRequest},
		// email: 全体は 254 文字まで
		{"email 254 chars", body(name, email254, password), http.StatusCreated},
		{"email 255 chars", body(name, email254[:64]+"@"+labelOf(63)+"."+labelOf(63)+"."+labelOf(62), password), http.StatusBadRequest},
		// email: @ の前は 64 文字まで
		{"email local part 64 chars", body(name, labelOf(64)+"@example.com", password), http.StatusCreated},
		{"email local part 65 chars", body(name, labelOf(65)+"@example.com", password), http.StatusBadRequest},
		// email: ドメインのラベルは 63 文字まで
		{"email domain label 63 chars", body(name, "taro@"+labelOf(63)+".com", password), http.StatusCreated},
		{"email domain label 64 chars", body(name, "taro@"+labelOf(64)+".com", password), http.StatusBadRequest},
		// password: 8〜72 文字、かつ 72 バイトまで
		{"password 7 chars", body(name, email, strings.Repeat("a", 7)), http.StatusBadRequest},
		{"password 8 chars", body(name, email, strings.Repeat("a", 8)), http.StatusCreated},
		{"password 72 chars", body(name, email, strings.Repeat("a", 72)), http.StatusCreated},
		{"password 73 chars", body(name, email, strings.Repeat("a", 73)), http.StatusBadRequest},
		{"password 72 bytes multibyte", body(name, email, strings.Repeat("あ", 24)), http.StatusCreated},
		{"password 73 bytes multibyte", body(name, email, strings.Repeat("あ", 24)+"a"), http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newTestServer(t)

			code, res := doJSON(t, srv, http.MethodPost, "/v1/signup", tt.body)
			if code != tt.want {
				t.Fatalf("status = %d, want %d, body = %v", code, tt.want, res)
			}
		})
	}
}

// ボディが上限（server.MaxBodyBytes）を超えたら、仕様書の検証より前に 413 を返す。
func TestSignup_BodyLimit(t *testing.T) {
	t.Parallel()

	// 末尾を空白で埋めて、指定したバイト数の正しい JSON にする
	bodyOfLen := func(n int) string {
		body := `{"name":"taro","email":"taro@example.com","password":"password123"}`
		return body + strings.Repeat(" ", n-len(body))
	}

	tests := []struct {
		name string
		body string
		want int
	}{
		{"just the limit", bodyOfLen(server.MaxBodyBytes), http.StatusCreated},
		{"one byte over the limit", bodyOfLen(server.MaxBodyBytes + 1), http.StatusRequestEntityTooLarge},
		{"far over the limit", bodyOfLen(10 * 1024 * 1024), http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newTestServer(t)

			code, res := doJSON(t, srv, http.MethodPost, "/v1/signup", tt.body)
			if code != tt.want {
				t.Fatalf("status = %d, want %d, body = %v", code, tt.want, res)
			}
		})
	}
}

// 仕様書の検証を通る最大のボディでも、上限に収まる。
func TestSignup_MaxValidBodyFitsLimit(t *testing.T) {
	t.Parallel()

	// 仕様書の maxLength は文字数で数えるため、1 文字あたりのバイト数が最大になるように各項目を作る。
	// - name: 形式の制限がないため、BMP 外の文字（😀）をサロゲートペアにエスケープすると 1 文字 12 バイトになる
	// - email: ASCII のみだが、\u0061 のようにエスケープすると 1 文字 6 バイトになる（254 文字）
	// - password: 72 バイトまでのため、1 バイト文字をエスケープして 72 文字（432 バイト）にしたときが最大
	escapeASCII := func(s string) string {
		var b strings.Builder
		for _, c := range []byte(s) {
			fmt.Fprintf(&b, `\u%04x`, c)
		}
		return b.String()
	}
	body := `{"name":"` + strings.Repeat(`\uD83D\uDE00`, 50) +
		`","email":"` + escapeASCII(email254) +
		`","password":"` + escapeASCII(labelOf(72)) + `"}`
	if len(body) > server.MaxBodyBytes {
		t.Fatalf("len(body) = %d, want <= %d", len(body), server.MaxBodyBytes)
	}
	t.Logf("max valid body = %d bytes", len(body))

	srv := newTestServer(t)
	code, res := doJSON(t, srv, http.MethodPost, "/v1/signup", body)
	if code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %v", code, http.StatusCreated, res)
	}
}

// メールアドレスの形式。想定するパターンは internal/domain/user.go の User.Email を参照。
func TestSignup_EmailFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		email string
		want  int
	}{
		{"taro@example.com", http.StatusCreated},
		{"Taro@Example.com", http.StatusCreated},
		{"taro+test@example.com", http.StatusCreated},
		{"taro.yamada@mail.example.co.jp", http.StatusCreated},
		{"taro@my-example.com", http.StatusCreated},
		{"1234@example.com", http.StatusCreated},
		{"!#$%&'*+/=?^_`{|}~-@example.com", http.StatusCreated},
		{"taro.yamada.jr@example.com", http.StatusCreated},

		{"taro@localhost", http.StatusBadRequest},
		{"山田@example.jp", http.StatusBadRequest},
		{"taro@例え.jp", http.StatusBadRequest},
		{"😀@example.com", http.StatusBadRequest},
		{"taro@😀.com", http.StatusBadRequest},
		{"ta ro@example.com", http.StatusBadRequest},
		{" taro@example.com", http.StatusBadRequest},
		{`\"taro\"@example.com`, http.StatusBadRequest},
		{"taro@[192.168.0.1]", http.StatusBadRequest},
		{"taro@@example.com", http.StatusBadRequest},
		{"taro", http.StatusBadRequest},
		{"@example.com", http.StatusBadRequest},
		{"taro@", http.StatusBadRequest},
		{"taro@-example.com", http.StatusBadRequest},
		{"taro@example-.com", http.StatusBadRequest},
		{"taro@example..com", http.StatusBadRequest},
		{"taro@example.com.", http.StatusBadRequest},
		{".taro@example.com", http.StatusBadRequest},
		{"taro.@example.com", http.StatusBadRequest},
		{"ta..ro@example.com", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			t.Parallel()
			srv := newTestServer(t)

			code, res := doJSON(t, srv, http.MethodPost, "/v1/signup",
				`{"name":"taro","email":"`+tt.email+`","password":"password123"}`)
			if code != tt.want {
				t.Fatalf("status = %d, want %d, body = %v", code, tt.want, res)
			}
		})
	}
}

func TestSignup_BadRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{"invalid email", `{"name":"taro","email":"not-an-email","password":"password123"}`},
		{"password too short", `{"name":"taro","email":"taro@example.com","password":"short"}`},
		{"password too long", `{"name":"taro","email":"taro@example.com","password":"` + strings.Repeat("a", 73) + `"}`},
		// 30 文字で仕様書の maxLength は満たすが、90 バイトで bcrypt の上限を超える
		{"multibyte password too long", `{"name":"taro","email":"taro@example.com","password":"` + strings.Repeat("あ", 30) + `"}`},
		// 制御文字。NUL は PostgreSQL に保存できないため、DB に届く前に 400 にする
		{"name with NUL", `{"name":"ta\u0000ro","email":"taro@example.com","password":"password123"}`},
		{"name with newline", `{"name":"ta\nro","email":"taro@example.com","password":"password123"}`},
		{"name with tab", `{"name":"ta\tro","email":"taro@example.com","password":"password123"}`},
		{"name with DEL", `{"name":"ta\u007fro","email":"taro@example.com","password":"password123"}`},
		{"email with NUL", `{"name":"taro","email":"ta\u0000ro@example.com","password":"password123"}`},
		{"empty name", `{"name":"","email":"taro@example.com","password":"password123"}`},
		{"missing name", `{"email":"taro@example.com","password":"password123"}`},
		{"malformed json", `{"name":`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newTestServer(t)

			code, body := doJSON(t, srv, http.MethodPost, "/v1/signup", tt.body)
			if code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body = %v", code, http.StatusBadRequest, body)
			}
			if body["message"] == nil {
				t.Errorf("body = %v, want message", body)
			}
		})
	}
}

// failingHandler は常に想定外のエラーを返すハンドラー。
type failingHandler struct{}

func (failingHandler) Signup(context.Context, oapi.SignupRequestObject) (oapi.SignupResponseObject, error) {
	return nil, errors.New("database is down")
}

// 想定外のエラーは 500 にし、エラーの詳細をクライアントに返さない。
func TestSignup_InternalServerError(t *testing.T) {
	t.Parallel()

	e, err := server.New(failingHandler{}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("server.New() error = %v", err)
	}

	code, body := doJSON(t, e, http.MethodPost, "/v1/signup",
		`{"name":"taro","email":"taro@example.com","password":"password123"}`)
	if code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body = %v", code, http.StatusInternalServerError, body)
	}
	if body["message"] != "Internal Server Error" {
		t.Errorf("message = %v, want %q", body["message"], "Internal Server Error")
	}
}
