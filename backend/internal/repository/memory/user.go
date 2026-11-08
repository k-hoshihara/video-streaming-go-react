// Package memory はメモリ上にデータを保持する repository の実装。
// DB を導入するまでの仮実装で、プロセスを再起動するとデータは消える。
package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/k-hoshihara/video-streaming-go-react/backend/internal/domain"
)

// UserRepository はユーザーをメモリ上に保持する。並行に呼び出しても安全。
type UserRepository struct {
	mu      sync.RWMutex
	byID    map[uuid.UUID]domain.User
	byEmail map[string]uuid.UUID // email -> id
}

// NewUserRepository は UserRepository を生成する。
func NewUserRepository() *UserRepository {
	return &UserRepository{
		byID:    make(map[uuid.UUID]domain.User),
		byEmail: make(map[string]uuid.UUID),
	}
}

// Create はユーザーを保存する。メールアドレスが登録済みなら domain.ErrEmailAlreadyExists を返す。
// 重複チェックと保存を同じロック内で行い、同時登録でも重複しないようにする。
func (r *UserRepository) Create(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byEmail[user.Email]; ok {
		return domain.ErrEmailAlreadyExists
	}
	r.byID[user.ID] = *user
	r.byEmail[user.Email] = user.ID
	return nil
}
