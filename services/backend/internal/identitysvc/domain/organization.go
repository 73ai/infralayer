package domain

import (
	"context"
	"time"

	"github.com/73ai/infralayer/services/backend"
	"github.com/google/uuid"
)

type OrganizationRepository interface {
	Create(context.Context, Organization) error
	OrganizationByClerkID(ctx context.Context, clerkOrgID string) (*Organization, error)
	OrganizationsByUserClerkID(ctx context.Context, clerkUserID string) ([]*Organization, error)
	Update(ctx context.Context, clerkOrgID string, org Organization) error
	SetMetadata(ctx context.Context, organizationID uuid.UUID, metadata OrganizationMetadata) error
	DeleteByClerkID(ctx context.Context, clerkOrgID string) error
}

type Organization struct {
	ID              uuid.UUID
	ClerkOrgID      string
	Name            string
	Slug            string
	CreatedByUserID uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Metadata        OrganizationMetadata
}

type OrganizationMetadata struct {
	OrganizationID     uuid.UUID
	CompanySize        backend.CompanySize
	TeamSize           backend.TeamSize
	UseCases           []backend.UseCase
	ObservabilityStack []backend.ObservabilityStack
	CompletedAt        time.Time
	UpdatedAt          time.Time
}
