// Command server は API サーバーを起動する。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/handler"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/repository/memory"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/repository/postgres"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/server"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	userRepo, closeRepo, err := newUserRepository(ctx, logger)
	if err != nil {
		return err
	}
	defer closeRepo()
	userUsecase := usecase.NewUserUsecase(userRepo)
	h := handler.New(userUsecase)

	e, err := server.New(h, logger)
	if err != nil {
		return err
	}

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server started", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// newUserRepository は DATABASE_URL が設定されていれば PostgreSQL、なければメモリ上の repository を返す。
func newUserRepository(ctx context.Context, logger *slog.Logger) (usecase.UserRepository, func(), error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		logger.Warn("DATABASE_URL is not set; using in-memory repository (data is lost on restart)")
		return memory.NewUserRepository(), func() {}, nil
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, nil, fmt.Errorf("create db pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("connect db: %w", err)
	}
	return postgres.NewUserRepository(pool), pool.Close, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
