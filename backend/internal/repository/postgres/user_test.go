package postgres

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/testdb"
)

func newUser(email string) *domain.User {
	return &domain.User{
		ID:           uuid.New(),
		Name:         "taro",
		Email:        email,
		PasswordHash: []byte("$2a$10$hash"),
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
}

func TestUserRepository_Create(t *testing.T) {
	t.Parallel()
	pool := testdb.New(t)
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

// 一意制約違反以外の DB エラーは ErrEmailAlreadyExists にせず、そのまま返す。
func TestUserRepository_Create_DBError(t *testing.T) {
	t.Parallel()
	pool := testdb.New(t)
	r := NewUserRepository(pool)
	pool.Close()

	err := r.Create(t.Context(), newUser("taro@example.com"))
	if err == nil || errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("Create() error = %v, want a non-duplicate error", err)
	}
}

// PostgreSQL が一意制約違反以外のエラー（ここではテーブルが存在しない）を返した場合も、
// ErrEmailAlreadyExists にせずそのまま返す。
func TestUserRepository_Create_OtherPgError(t *testing.T) {
	t.Parallel()
	pool := testdb.New(t)
	r := NewUserRepository(pool)
	if _, err := pool.Exec(t.Context(), "DROP TABLE users"); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	err := r.Create(t.Context(), newUser("taro@example.com"))
	if _, ok := errors.AsType[*pgconn.PgError](err); !ok {
		t.Fatalf("Create() error = %v, want a PostgreSQL error", err)
	}
	if errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("Create() error = %v, must not be %v", err, domain.ErrEmailAlreadyExists)
	}
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	t.Parallel()
	r := NewUserRepository(testdb.New(t))

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
	r := NewUserRepository(testdb.New(t))

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
