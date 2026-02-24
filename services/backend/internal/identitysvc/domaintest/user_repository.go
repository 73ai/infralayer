package domaintest

import (
	"context"
	"fmt"
	"sync"

	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"
)

type userRepository struct {
	mu    sync.RWMutex
	users map[string]domain.User
}

func NewUserRepository() domain.UserRepository {
	return &userRepository{
		users: make(map[string]domain.User),
	}
}

func (r *userRepository) Create(ctx context.Context, user domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ClerkUserID]; exists {
		return fmt.Errorf("user with clerk_user_id %s already exists", user.ClerkUserID)
	}

	r.users[user.ClerkUserID] = user
	return nil
}

func (r *userRepository) UserByClerkID(ctx context.Context, clerkUserID string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[clerkUserID]
	if !exists {
		return nil, fmt.Errorf("user with clerk_user_id %s not found", clerkUserID)
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, clerkUserID string, user domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[clerkUserID]; !exists {
		return fmt.Errorf("user with clerk_user_id %s not found", clerkUserID)
	}

	existing := r.users[clerkUserID]
	existing.Email = user.Email
	existing.FirstName = user.FirstName
	existing.LastName = user.LastName
	r.users[clerkUserID] = existing

	return nil
}

func (r *userRepository) DeleteByClerkID(ctx context.Context, clerkUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[clerkUserID]; !exists {
		return fmt.Errorf("user with clerk_user_id %s not found", clerkUserID)
	}

	delete(r.users, clerkUserID)
	return nil
}
