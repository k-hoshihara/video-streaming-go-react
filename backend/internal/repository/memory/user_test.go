package memory

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
)

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	t.Parallel()

	r := NewUserRepository()
	if err := r.Create(t.Context(), &domain.User{ID: uuid.New(), Email: "taro@example.com"}); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	err := r.Create(t.Context(), &domain.User{ID: uuid.New(), Email: "taro@example.com"})
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("second Create() error = %v, want %v", err, domain.ErrEmailAlreadyExists)
	}
}

// 同じメールアドレスで同時に登録しても、成功するのは 1 件だけであることを確認する。
func TestUserRepository_Create_Concurrent(t *testing.T) {
	t.Parallel()

	r := NewUserRepository()
	const n = 50
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
	)
	for range n {
		wg.Go(func() {
			err := r.Create(t.Context(), &domain.User{ID: uuid.New(), Email: "taro@example.com"})
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
