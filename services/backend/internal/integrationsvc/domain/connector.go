package domain

import (
	"context"

	"github.com/73ai/infralayer/services/backend"
	"github.com/google/uuid"
)

type Connector interface {
	// Authorization methods
	InitiateAuthorization(organizationID string, userID string) (backend.IntegrationAuthorizationIntent, error)
	ParseState(state string) (organizationID uuid.UUID, userID uuid.UUID, err error)
	CompleteAuthorization(authData backend.AuthorizationData) (backend.Credentials, error)
	ValidateCredentials(creds backend.Credentials) error
	RefreshCredentials(creds backend.Credentials) (backend.Credentials, error)
	RevokeCredentials(creds backend.Credentials) error

	// Webhook methods
	ConfigureWebhooks(integrationID string, creds backend.Credentials) error
	ValidateWebhookSignature(payload []byte, signature string, secret string) error

	// Event subscription method - each connector handles its own communication
	Subscribe(ctx context.Context, handler func(ctx context.Context, event any) error) error

	// Event processing method - each connector processes its own events
	ProcessEvent(ctx context.Context, event any) error

	// Sync method - performs connector-specific synchronization operations
	Sync(ctx context.Context, integration backend.Integration, params map[string]string) error
}
