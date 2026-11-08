package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/handler"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/oapi"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/repository/memory"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/server"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/usecase"
)

// newTestServer は本番と同じ構成（仕様書による検証を含む）で組み立てたサーバーを返す。
func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	h := handler.New(usecase.NewUserUsecase(memory.NewUserRepository()))
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
