package domain

import (
	"context"

	"github.com/73ai/infralayer/services/backend"
	"github.com/google/uuid"
)

type Integration struct {
	backend.Integration
	BusinessID        string
	ProviderProjectID string
}

type IntegrationRepository interface {
	Integrations(ctx context.Context, businessID uuid.UUID) ([]Integration, error)
	SaveIntegration(ctx context.Context, integration Integration) error
}
