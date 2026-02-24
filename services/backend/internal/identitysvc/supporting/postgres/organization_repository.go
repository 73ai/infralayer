package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type organizationRepository struct {
	db      *sql.DB
	queries *Queries
}

func NewOrganizationRepository(sqlDB *sql.DB) domain.OrganizationRepository {
	return &organizationRepository{
		db:      sqlDB,
		queries: New(sqlDB),
	}
}

func (r *organizationRepository) Create(ctx context.Context, org domain.Organization) error {
	err := r.queries.CreateOrganization(ctx, CreateOrganizationParams{
		ClerkOrgID:      org.ClerkOrgID,
		Name:            org.Name,
		Slug:            org.Slug,
		CreatedByUserID: uuid.NullUUID{UUID: org.CreatedByUserID, Valid: true},
	})

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrDuplicateKey
		}
		return err
	}

	return nil
}

func (r *organizationRepository) OrganizationByClerkID(ctx context.Context, clerkOrgID string) (*domain.Organization, error) {
	org, err := r.queries.GetOrganizationByClerkID(ctx, clerkOrgID)
	if err != nil {
		return nil, fmt.Errorf("organization by clerk id: %w", err)
	}

	result := &domain.Organization{
		ID:              org.ID,
		ClerkOrgID:      org.ClerkOrgID,
		Name:            org.Name,
		Slug:            org.Slug,
		CreatedByUserID: org.CreatedByUserID.UUID,
		CreatedAt:       org.CreatedAt.Time,
		UpdatedAt:       org.UpdatedAt.Time,
	}

	metadata, err := r.queries.GetOrganizationMetadataByOrganizationID(ctx, org.ID)
	if err == nil {
		useCases := make([]backend.UseCase, len(metadata.UseCases))
		for i, uc := range metadata.UseCases {
			useCases[i] = backend.UseCase(uc)
		}

		stack := make([]backend.ObservabilityStack, len(metadata.ObservabilityStack))
		for i, s := range metadata.ObservabilityStack {
			stack[i] = backend.ObservabilityStack(s)
		}

		result.Metadata = domain.OrganizationMetadata{
			OrganizationID:     metadata.OrganizationID,
			CompanySize:        backend.CompanySize(metadata.CompanySize),
			TeamSize:           backend.TeamSize(metadata.TeamSize),
			UseCases:           useCases,
			ObservabilityStack: stack,
			CompletedAt:        metadata.CompletedAt.Time,
			UpdatedAt:          metadata.UpdatedAt.Time,
		}
	}

	return result, nil
}

func (r *organizationRepository) OrganizationsByUserClerkID(ctx context.Context, clerkUserID string) ([]*domain.Organization, error) {
	orgs, err := r.queries.GetOrganizationsByUserClerkID(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Organization, len(orgs))
	for i, org := range orgs {
		result[i] = &domain.Organization{
			ID:              org.ID,
			ClerkOrgID:      org.ClerkOrgID,
			Name:            org.Name,
			Slug:            org.Slug,
			CreatedByUserID: org.CreatedByUserID.UUID,
			CreatedAt:       org.CreatedAt.Time,
			UpdatedAt:       org.UpdatedAt.Time,
		}

		metadata, err := r.queries.GetOrganizationMetadataByOrganizationID(ctx, org.ID)
		if err == nil {
			useCases := make([]backend.UseCase, len(metadata.UseCases))
			for j, uc := range metadata.UseCases {
				useCases[j] = backend.UseCase(uc)
			}

			stack := make([]backend.ObservabilityStack, len(metadata.ObservabilityStack))
			for j, s := range metadata.ObservabilityStack {
				stack[j] = backend.ObservabilityStack(s)
			}

			result[i].Metadata = domain.OrganizationMetadata{
				OrganizationID:     metadata.OrganizationID,
				CompanySize:        backend.CompanySize(metadata.CompanySize),
				TeamSize:           backend.TeamSize(metadata.TeamSize),
				UseCases:           useCases,
				ObservabilityStack: stack,
				CompletedAt:        metadata.CompletedAt.Time,
				UpdatedAt:          metadata.UpdatedAt.Time,
			}
		}
	}

	return result, nil
}

func (r *organizationRepository) Update(ctx context.Context, clerkOrgID string, org domain.Organization) error {
	return r.queries.UpdateOrganization(ctx, UpdateOrganizationParams{
		ClerkOrgID: clerkOrgID,
		Name:       org.Name,
		Slug:       org.Slug,
	})
}

func (r *organizationRepository) SetMetadata(ctx context.Context, organizationID uuid.UUID, metadata domain.OrganizationMetadata) error {
	useCases := make([]string, len(metadata.UseCases))
	for i, uc := range metadata.UseCases {
		useCases[i] = string(uc)
	}

	stack := make([]string, len(metadata.ObservabilityStack))
	for i, s := range metadata.ObservabilityStack {
		stack[i] = string(s)
	}

	existing, err := r.queries.GetOrganizationMetadataByOrganizationID(ctx, organizationID)
	if err != nil {
		return r.queries.CreateOrganizationMetadata(ctx, CreateOrganizationMetadataParams{
			OrganizationID:     organizationID,
			CompanySize:        string(metadata.CompanySize),
			TeamSize:           string(metadata.TeamSize),
			UseCases:           useCases,
			ObservabilityStack: stack,
		})
	}

	_ = existing
	return r.queries.UpdateOrganizationMetadata(ctx, UpdateOrganizationMetadataParams{
		OrganizationID:     organizationID,
		CompanySize:        string(metadata.CompanySize),
		TeamSize:           string(metadata.TeamSize),
		UseCases:           useCases,
		ObservabilityStack: stack,
	})
}

func (r *organizationRepository) DeleteByClerkID(ctx context.Context, clerkOrgID string) error {

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()
	qtx := r.queries.WithTx(tx)

	org, err := qtx.GetOrganizationByClerkID(ctx, clerkOrgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("organization with clerk_org_id %s not found", clerkOrgID)
		}
		return fmt.Errorf("error fetching organization: %w", err)
	}
	err = qtx.DeleteOrganizationMetadataByOrganizationID(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("error deleting organization metadata: %w", err)
	}
	err = qtx.DeleteOrganizationByClerkID(ctx, clerkOrgID)
	if err != nil {
		return fmt.Errorf("error deleting organization: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}
