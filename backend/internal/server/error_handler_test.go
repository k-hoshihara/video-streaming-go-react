package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// クライアント側のエラー（4xx）はメッセージをそのまま返す。
// サーバー側のエラー（5xx）と想定外のエラーは詳細を返さず、元のエラーをログに出す。
func TestErrorHandler(t *testing.T) {
	t.Parallel()

	internal := errors.New("db password is wrong")
	tests := []struct {
		name        string
		err         error
		wantCode    int
		wantMessage string
		wantLogged  bool
	}{
		{"4xx HTTPError", echo.NewHTTPError(http.StatusBadRequest, "name is required"),
			http.StatusBadRequest, "name is required", false},
		// 4xx と 5xx の境界
		{"499 HTTPError", echo.NewHTTPError(499, "client closed request"),
			499, "client closed request", false},
		{"500 HTTPError", echo.NewHTTPError(http.StatusInternalServerError, "error validating request: secret").SetInternal(internal),
			http.StatusInternalServerError, "Internal Server Error", true},
		{"503 HTTPError", echo.NewHTTPError(http.StatusServiceUnavailable, "db is down: secret").SetInternal(internal),
			http.StatusServiceUnavailable, "Service Unavailable", true},
		{"wrapped 5xx HTTPError", fmt.Errorf("wrap: %w", echo.NewHTTPError(http.StatusBadGateway, "secret")),
			http.StatusBadGateway, "Bad Gateway", true},
		{"unexpected error", fmt.Errorf("query: %w", internal),
			http.StatusInternalServerError, "Internal Server Error", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs []string
			logger := slog.New(&recordHandler{onHandle: func(r slog.Record) {
				r.Attrs(func(a slog.Attr) bool {
					logs = append(logs, a.Value.String())
					return true
				})
			}})
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil), rec)

			errorHandler(logger)(tt.err, c)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			var body struct{ Message string }
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v, body = %s", err, rec.Body)
			}
			if body.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", body.Message, tt.wantMessage)
			}
			if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), internal.Error()) {
				t.Errorf("response leaks error details: %s", rec.Body)
			}

			logged := strings.Join(logs, "\n")
			if tt.wantLogged && !strings.Contains(logged, tt.err.Error()) {
				t.Errorf("log = %q, want it to contain %q", logged, tt.err.Error())
			}
			if !tt.wantLogged && len(logs) != 0 {
				t.Errorf("log = %q, want nothing", logged)
			}
		})
	}
}

// レスポンスを送信済みなら、何も書き込まない。
func TestErrorHandler_Committed(t *testing.T) {
	t.Parallel()

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil), rec)
	if err := c.String(http.StatusOK, "done"); err != nil {
		t.Fatal(err)
	}

	errorHandler(slog.New(slog.DiscardHandler))(errors.New("late error"), c)

	if rec.Code != http.StatusOK || rec.Body.String() != "done" {
		t.Errorf("response = %d %q, want 200 %q", rec.Code, rec.Body.String(), "done")
	}
}

// failingWriter は書き込みに必ず失敗する ResponseWriter。
type failingWriter struct{ *httptest.ResponseRecorder }

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("connection reset") }

// レスポンスの書き込みに失敗しても panic せず、ログに出す。
func TestErrorHandler_WriteFailure(t *testing.T) {
	t.Parallel()

	var logged bool
	logger := slog.New(&recordHandler{onHandle: func(r slog.Record) {
		if r.Message == "write error response" {
			logged = true
		}
	}})

	e := echo.New()
	c := e.NewContext(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil),
		failingWriter{httptest.NewRecorder()})

	errorHandler(logger)(echo.NewHTTPError(http.StatusBadRequest, "bad"), c)

	if !logged {
		t.Error("write failure was not logged")
	}
}

// recordHandler はログを受け取ってコールバックを呼ぶ slog.Handler。
type recordHandler struct {
	onHandle func(slog.Record)
}

func (h *recordHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *recordHandler) Handle(_ context.Context, r slog.Record) error {
	h.onHandle(r)
	return nil
}

func (h *recordHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordHandler) WithGroup(string) slog.Handler { return h }
