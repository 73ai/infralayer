package domain

import (
	"context"

	"github.com/73ai/infralayer/services/backend"
	"github.com/google/uuid"
)

type IntegrationRepository interface {
	Store(ctx context.Context, integration backend.Integration) error
	Update(ctx context.Context, integration backend.Integration) error
	FindByID(ctx context.Context, id uuid.UUID) (backend.Integration, error)
	FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]backend.Integration, error)
	FindByOrganizationAndType(ctx context.Context, orgID uuid.UUID, connectorType backend.ConnectorType) ([]backend.Integration, error)
	FindByOrganizationAndStatus(ctx context.Context, orgID uuid.UUID, status backend.IntegrationStatus) ([]backend.Integration, error)
	FindByOrganizationTypeAndStatus(ctx context.Context, orgID uuid.UUID, connectorType backend.ConnectorType, status backend.IntegrationStatus) ([]backend.Integration, error)
	FindByBotIDAndType(ctx context.Context, botID string, connectorType backend.ConnectorType) (backend.Integration, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status backend.IntegrationStatus) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	UpdateMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
