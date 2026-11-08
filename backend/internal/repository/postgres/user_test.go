package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
)

// newTestPool はテストごとに専用のスキーマを作ってマイグレーションを適用し、そのスキーマに接続するプールを返す。
// TEST_DATABASE_URL が未設定ならテストをスキップする。
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := t.Context()

	// 英数字だけの名前にして、SQL にそのまま埋め込めるようにする
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)

	files, err := filepath.Glob("../../../db/migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("find migrations: %v (found %d)", err, len(files))
	}
	for _, f := range files {
		sql, err := os.ReadFile(f) //nolint:gosec // テスト用の固定パス
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s: %v", f, err)
		}
	}
	return pool
}

func newUser(email string) *domain.User {
	return &domain.User{
		ID:           uuid.NewString(),
		Name:         "taro",
		Email:        email,
		PasswordHash: []byte("$2a$10$hash"),
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
}

func TestUserRepository_Create(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t)
	r := NewUserRepository(pool)

	want := newUser("taro@example.com")
	if err := r.Create(t.Context(), want); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var got domain.User
	var hash string
	err := pool.QueryRow(t.Context(),
		`SELECT id, name, email, password_hash, created_at FROM users WHERE id = $1`, want.ID,
	).Scan(&got.ID, &got.Name, &got.Email, &hash, &got.CreatedAt)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name || got.Email != want.Email ||
		hash != string(want.PasswordHash) || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("stored = %+v (hash %q), want %+v", got, hash, want)
	}
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	t.Parallel()
	r := NewUserRepository(newTestPool(t))

	if err := r.Create(t.Context(), newUser("taro@example.com")); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	err := r.Create(t.Context(), newUser("taro@example.com"))
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("second Create() error = %v, want %v", err, domain.ErrEmailAlreadyExists)
	}
}

// 同じメールアドレスで同時に登録しても、成功するのは 1 件だけであることを確認する。
func TestUserRepository_Create_Concurrent(t *testing.T) {
	t.Parallel()
	r := NewUserRepository(newTestPool(t))

	const n = 20
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
	)
	for range n {
		wg.Go(func() {
			err := r.Create(t.Context(), newUser("taro@example.com"))
			if err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			} else if !errors.Is(err, domain.ErrEmailAlreadyExists) {
				t.Errorf("Create() unexpected error = %v", err)
			}
		})
	}
	wg.Wait()

	if succeeded != 1 {
		t.Fatalf("succeeded = %d, want 1", succeeded)
	}
}
