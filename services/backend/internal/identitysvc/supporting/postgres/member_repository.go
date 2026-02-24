package postgres

import (
	"context"
	"database/sql"

	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type memberRepository struct {
	queries *Queries
}

func NewMemberRepository(sqlDB *sql.DB) domain.MemberRepository {
	return &memberRepository{
		queries: New(sqlDB),
	}
}

func (r *memberRepository) Create(ctx context.Context, member domain.OrganizationMember) error {
	err := r.queries.CreateOrganizationMember(ctx, CreateOrganizationMemberParams{
		UserID:         member.UserID,
		OrganizationID: member.OrganizationID,
		ClerkUserID:    member.ClerkUserID,
		ClerkOrgID:     member.ClerkOrgID,
		Role:           member.Role,
	})

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrDuplicateKey
		}
		return err
	}

	return nil
}

func (r *memberRepository) DeleteByClerkIDs(ctx context.Context, clerkUserID string, clerkOrgID string) error {
	return r.queries.DeleteOrganizationMemberByClerkIDs(ctx, DeleteOrganizationMemberByClerkIDsParams{
		ClerkUserID: clerkUserID,
		ClerkOrgID:  clerkOrgID,
	})
}

func (r *memberRepository) MembersByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]*domain.OrganizationMember, error) {
	members, err := r.queries.GetOrganizationMembersByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.OrganizationMember, len(members))
	for i, member := range members {
		result[i] = &domain.OrganizationMember{
			UserID:         member.UserID,
			OrganizationID: member.OrganizationID,
			ClerkUserID:    member.ClerkUserID,
			ClerkOrgID:     member.ClerkOrgID,
			Role:           member.Role,
			JoinedAt:       member.JoinedAt.Time,
		}
	}

	return result, nil
}

func (r *memberRepository) MembersByUserClerkID(ctx context.Context, clerkUserID string) ([]*domain.OrganizationMember, error) {
	members, err := r.queries.GetOrganizationMembersByUserClerkID(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.OrganizationMember, len(members))
	for i, member := range members {
		result[i] = &domain.OrganizationMember{
			UserID:         member.UserID,
			OrganizationID: member.OrganizationID,
			ClerkUserID:    member.ClerkUserID,
			ClerkOrgID:     member.ClerkOrgID,
			Role:           member.Role,
			JoinedAt:       member.JoinedAt.Time,
		}
	}

	return result, nil
}

func (r *memberRepository) UpdateByClerkIDs(ctx context.Context, clerkUserID string, clerkOrgID string, role string) error {
	return r.queries.UpdateOrganizationMemberByClerkIDs(ctx, UpdateOrganizationMemberByClerkIDsParams{
		ClerkUserID: clerkUserID,
		ClerkOrgID:  clerkOrgID,
		Role:        role,
	})
}
