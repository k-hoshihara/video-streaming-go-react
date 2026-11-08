// Package testdb は、テストで PostgreSQL を使うためのヘルパーを提供する。
// テストからだけ使う。
package testdb

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// New はテストごとに専用のスキーマを作ってマイグレーションを適用し、そのスキーマに接続するプールを返す。
// スキーマとプールはテストの終了時に片付ける。
//
// TEST_DATABASE_URL が未設定の場合、ローカルではテストをスキップする。
// CI（環境変数 CI が設定されている）では、DB のテストが黙ってスキップされないよう失敗にする。
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL is not set")
		}
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

	files, err := filepath.Glob(filepath.Join(migrationsDir(), "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("find migrations: %v (found %d)", err, len(files))
	}
	for _, f := range files {
		sql, err := os.ReadFile(f) //nolint:gosec // リポジトリ内の固定のディレクトリ
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s: %v", f, err)
		}
	}
	return pool
}

// migrationsDir は backend/db/migrations の絶対パスを返す。
// どのパッケージのテストから呼ばれても同じ場所を指すよう、このファイルの位置を基準にする。
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "db", "migrations")
}
