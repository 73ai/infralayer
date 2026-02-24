package postgres

import (
	"context"
	"database/sql"

	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"
	"github.com/lib/pq"
)

type userRepository struct {
	queries *Queries
}

func NewUserRepository(sqlDB *sql.DB) domain.UserRepository {
	return &userRepository{
		queries: New(sqlDB),
	}
}

func (r *userRepository) Create(ctx context.Context, user domain.User) error {
	err := r.queries.CreateUser(ctx, CreateUserParams{
		ClerkUserID: user.ClerkUserID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
	})

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrDuplicateKey
		}
		return err
	}

	return nil
}

func (r *userRepository) UserByClerkID(ctx context.Context, clerkUserID string) (*domain.User, error) {
	user, err := r.queries.GetUserByClerkID(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:          user.ID,
		ClerkUserID: user.ClerkUserID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		CreatedAt:   user.CreatedAt.Time,
		UpdatedAt:   user.UpdatedAt.Time,
	}, nil
}

func (r *userRepository) Update(ctx context.Context, clerkUserID string, user domain.User) error {
	return r.queries.UpdateUser(ctx, UpdateUserParams{
		ClerkUserID: clerkUserID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
	})
}

func (r *userRepository) DeleteByClerkID(ctx context.Context, clerkUserID string) error {
	return r.queries.DeleteUserByClerkID(ctx, clerkUserID)
}
