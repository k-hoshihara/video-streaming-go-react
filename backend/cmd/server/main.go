// Command server は API サーバーを起動する。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/handler"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/repository/postgres"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/server"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, logger)
	stop()
	if err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// run は DB に接続してサーバーを起動し、ctx が終了したら停止する。
func run(ctx context.Context, logger *slog.Logger) error {
	pool, err := newDBPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	h := handler.New(usecase.NewUserUsecase(postgres.NewUserRepository(pool)))
	e, err := server.New(h, logger)
	if err != nil {
		return err
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", ":"+envOr("PORT", "8080"))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	srv := &http.Server{
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
		// ボディを少しずつ送り続ける接続を切るため、ボディを含むリクエスト全体の読み込みにも上限を設ける
		ReadTimeout: 30 * time.Second,
	}
	logger.Info("server started", "addr", ln.Addr().String())
	return serve(ctx, srv, ln, logger)
}

// serve は ln でリクエストを受け付け、ctx が終了したら処理中のリクエストを待ってから停止する。
func serve(ctx context.Context, srv *http.Server, ln net.Listener, logger *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		// Serve は Shutdown を呼ぶまで戻らず、戻るときは必ずエラーを返す。
		errCh <- srv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// newDBPool は DATABASE_URL の PostgreSQL に接続する。
func newDBPool(ctx context.Context) (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect db: %w", err)
	}
	return pool, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
