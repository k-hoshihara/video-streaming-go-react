package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/testdb"
)

var discard = slog.New(slog.DiscardHandler)

// t.Setenv を使うテストは並列に実行できないため、t.Parallel は付けない。

func TestEnvOr(t *testing.T) {
	t.Setenv("TEST_ENV_OR", "")
	if got := envOr("TEST_ENV_OR", "fallback"); got != "fallback" {
		t.Errorf("unset: envOr() = %q, want %q", got, "fallback")
	}

	t.Setenv("TEST_ENV_OR", "value")
	if got := envOr("TEST_ENV_OR", "fallback"); got != "value" {
		t.Errorf("set: envOr() = %q, want %q", got, "value")
	}
}

func TestNewDBPool(t *testing.T) {
	url := testdb.URL(t)
	t.Setenv("DATABASE_URL", url)

	pool, err := newDBPool(t.Context())
	if err != nil {
		t.Fatalf("newDBPool() error = %v", err)
	}
	pool.Close()
}

func TestNewDBPool_Errors(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"DATABASE_URL is not set", ""},
		{"invalid DATABASE_URL", "host=localhost port=not-a-number"},
		// 接続先に DB がないため Ping が失敗する
		{"cannot connect", "postgres://app:app@127.0.0.1:1/app?sslmode=disable&connect_timeout=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tt.url)

			pool, err := newDBPool(t.Context())
			if err == nil {
				pool.Close()
				t.Fatal("newDBPool() error = nil, want an error")
			}
		})
	}
}

// 起動したサーバーがリクエストに応答し、ctx の終了で正常に停止することを確認する。
func TestRun(t *testing.T) {
	t.Setenv("DATABASE_URL", testdb.URL(t))
	port := freePort(t)
	t.Setenv("PORT", strconv.Itoa(port))

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- run(ctx, discard) }()

	// 未定義のパスには 404 を返す。応答があればサーバーは起動している。
	waitForServer(t, "http://127.0.0.1:"+strconv.Itoa(port)+"/not-found", done)

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
}

func TestRun_DBError(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if err := run(t.Context(), discard); err == nil {
		t.Fatal("run() error = nil, want an error")
	}
}

// ポートが使用中なら、起動に失敗する。
func TestRun_ListenError(t *testing.T) {
	t.Setenv("DATABASE_URL", testdb.URL(t))
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	t.Setenv("PORT", strconv.Itoa(ln.Addr().(*net.TCPAddr).Port))

	if err := run(t.Context(), discard); err == nil {
		t.Fatal("run() error = nil, want an error")
	}
}

// リクエストの受け付けに失敗したら、エラーを返す。
func TestServe_Error(t *testing.T) {
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	if err := serve(t.Context(), &http.Server{ReadHeaderTimeout: time.Second}, ln, discard); err == nil {
		t.Fatal("serve() error = nil, want an error")
	}
}

// freePort は空いているポート番号を返す。
func freePort(t *testing.T) int {
	t.Helper()

	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	return ln.Addr().(*net.TCPAddr).Port
}

// waitForServer はサーバーが応答するまで待つ。run が先に終了したら失敗にする。
func waitForServer(t *testing.T, url string, done <-chan error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res, err := http.DefaultClient.Do(req); err == nil {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
			return
		}

		select {
		case err := <-done:
			t.Fatalf("run() stopped before the server responded: %v", err)
		case <-ctx.Done():
			t.Fatalf("server did not respond: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}
