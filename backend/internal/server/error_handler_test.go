package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

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
