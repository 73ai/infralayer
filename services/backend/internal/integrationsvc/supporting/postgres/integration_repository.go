package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

type integrationRepository struct {
	queries *Queries
}

func NewIntegrationRepository(sqlDB *sql.DB) domain.IntegrationRepository {
	return &integrationRepository{
		queries: New(sqlDB),
	}
}

func (r *integrationRepository) Store(ctx context.Context, integration backend.Integration) error {
	metadata := make(map[string]any)
	for k, v := range integration.Metadata {
		metadata[k] = v
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	integrationID := integration.ID
	organizationID := integration.OrganizationID
	userID := integration.UserID

	var botID sql.NullString
	if integration.BotID != "" {
		botID = sql.NullString{String: integration.BotID, Valid: true}
	}

	var connectorUserID sql.NullString
	if integration.ConnectorUserID != "" {
		connectorUserID = sql.NullString{String: integration.ConnectorUserID, Valid: true}
	}

	var connectorOrganizationID sql.NullString
	if integration.ConnectorOrganizationID != "" {
		connectorOrganizationID = sql.NullString{String: integration.ConnectorOrganizationID, Valid: true}
	}

	var lastUsedAt sql.NullTime
	if integration.LastUsedAt != nil {
		lastUsedAt = sql.NullTime{Time: *integration.LastUsedAt, Valid: true}
	}

	return r.queries.StoreIntegration(ctx, StoreIntegrationParams{
		ID:                      integrationID,
		OrganizationID:          organizationID,
		UserID:                  userID,
		ConnectorType:           string(integration.ConnectorType),
		Status:                  string(integration.Status),
		BotID:                   botID,
		ConnectorUserID:         connectorUserID,
		ConnectorOrganizationID: connectorOrganizationID,
		Metadata:                pqtype.NullRawMessage{RawMessage: metadataJSON, Valid: true},
		CreatedAt:               integration.CreatedAt,
		UpdatedAt:               integration.UpdatedAt,
		LastUsedAt:              lastUsedAt,
	})
}

func (r *integrationRepository) Update(ctx context.Context, integration backend.Integration) error {
	metadata := make(map[string]any)
	for k, v := range integration.Metadata {
		metadata[k] = v
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	integrationID := integration.ID

	var botID sql.NullString
	if integration.BotID != "" {
		botID = sql.NullString{String: integration.BotID, Valid: true}
	}

	var connectorUserID sql.NullString
	if integration.ConnectorUserID != "" {
		connectorUserID = sql.NullString{String: integration.ConnectorUserID, Valid: true}
	}

	var connectorOrganizationID sql.NullString
	if integration.ConnectorOrganizationID != "" {
		connectorOrganizationID = sql.NullString{String: integration.ConnectorOrganizationID, Valid: true}
	}

	var lastUsedAt sql.NullTime
	if integration.LastUsedAt != nil {
		lastUsedAt = sql.NullTime{Time: *integration.LastUsedAt, Valid: true}
	}

	return r.queries.UpdateIntegration(ctx, UpdateIntegrationParams{
		ID:                      integrationID,
		ConnectorType:           string(integration.ConnectorType),
		Status:                  string(integration.Status),
		BotID:                   botID,
		ConnectorUserID:         connectorUserID,
		ConnectorOrganizationID: connectorOrganizationID,
		Metadata:                pqtype.NullRawMessage{RawMessage: metadataJSON, Valid: true},
		UpdatedAt:               integration.UpdatedAt,
		LastUsedAt:              lastUsedAt,
	})
}

func (r *integrationRepository) FindByID(ctx context.Context, id uuid.UUID) (backend.Integration, error) {
	dbIntegration, err := r.queries.FindIntegrationByID(ctx, id)
	if err != nil {
		return backend.Integration{}, fmt.Errorf("failed to find integration: %w", err)
	}

	return r.toSpecIntegration(dbIntegration)
}

func (r *integrationRepository) FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]backend.Integration, error) {
	dbIntegrations, err := r.queries.FindIntegrationsByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to find integrations: %w", err)
	}

	integrations := make([]backend.Integration, len(dbIntegrations))
	for i, dbIntegration := range dbIntegrations {
		integration, err := r.toSpecIntegration(dbIntegration)
		if err != nil {
			return nil, fmt.Errorf("failed to map integration: %w", err)
		}
		integrations[i] = integration
	}

	return integrations, nil
}

func (r *integrationRepository) FindByOrganizationAndType(ctx context.Context, orgID uuid.UUID, connectorType backend.ConnectorType) ([]backend.Integration, error) {
	dbIntegrations, err := r.queries.FindIntegrationsByOrganizationAndType(ctx, FindIntegrationsByOrganizationAndTypeParams{
		OrganizationID: orgID,
		ConnectorType:  string(connectorType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find integrations: %w", err)
	}

	integrations := make([]backend.Integration, len(dbIntegrations))
	for i, dbIntegration := range dbIntegrations {
		integration, err := r.toSpecIntegration(dbIntegration)
		if err != nil {
			return nil, fmt.Errorf("failed to map integration: %w", err)
		}
		integrations[i] = integration
	}

	return integrations, nil
}

func (r *integrationRepository) FindByOrganizationAndStatus(ctx context.Context, orgID uuid.UUID, status backend.IntegrationStatus) ([]backend.Integration, error) {
	dbIntegrations, err := r.queries.FindIntegrationsByOrganizationAndStatus(ctx, FindIntegrationsByOrganizationAndStatusParams{
		OrganizationID: orgID,
		Status:         string(status),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find integrations: %w", err)
	}

	integrations := make([]backend.Integration, len(dbIntegrations))
	for i, dbIntegration := range dbIntegrations {
		integration, err := r.toSpecIntegration(dbIntegration)
		if err != nil {
			return nil, fmt.Errorf("failed to map integration: %w", err)
		}
		integrations[i] = integration
	}

	return integrations, nil
}

func (r *integrationRepository) FindByOrganizationTypeAndStatus(ctx context.Context, orgID uuid.UUID, connectorType backend.ConnectorType, status backend.IntegrationStatus) ([]backend.Integration, error) {
	dbIntegrations, err := r.queries.FindIntegrationsByOrganizationTypeAndStatus(ctx, FindIntegrationsByOrganizationTypeAndStatusParams{
		OrganizationID: orgID,
		ConnectorType:  string(connectorType),
		Status:         string(status),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find integrations: %w", err)
	}

	integrations := make([]backend.Integration, len(dbIntegrations))
	for i, dbIntegration := range dbIntegrations {
		integration, err := r.toSpecIntegration(dbIntegration)
		if err != nil {
			return nil, fmt.Errorf("failed to map integration: %w", err)
		}
		integrations[i] = integration
	}

	return integrations, nil
}

func (r *integrationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status backend.IntegrationStatus) error {
	return r.queries.UpdateIntegrationStatus(ctx, UpdateIntegrationStatusParams{
		ID:     id,
		Status: string(status),
	})
}

func (r *integrationRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return r.queries.UpdateIntegrationLastUsed(ctx, id)
}

func (r *integrationRepository) UpdateMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error {
	metadataMap := make(map[string]any)
	for k, v := range metadata {
		metadataMap[k] = v
	}

	metadataJSON, err := json.Marshal(metadataMap)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return r.queries.UpdateIntegrationMetadata(ctx, UpdateIntegrationMetadataParams{
		ID:       id,
		Metadata: pqtype.NullRawMessage{RawMessage: metadataJSON, Valid: true},
	})
}

func (r *integrationRepository) Delete(ctx context.Context, id uuid.UUID) error {

	return r.queries.DeleteIntegration(ctx, id)
}

func (r *integrationRepository) FindByBotIDAndType(ctx context.Context, botID string, connectorType backend.ConnectorType) (backend.Integration, error) {
	dbIntegration, err := r.queries.FindIntegrationByBotIDAndType(ctx, FindIntegrationByBotIDAndTypeParams{
		BotID:         sql.NullString{String: botID, Valid: true},
		ConnectorType: string(connectorType),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return backend.Integration{}, domain.ErrIntegrationNotFound
		}
		return backend.Integration{}, fmt.Errorf("failed to find integration by bot ID: %w", err)
	}

	return r.toSpecIntegration(dbIntegration)
}

func (r *integrationRepository) toSpecIntegration(dbIntegration Integration) (backend.Integration, error) {
	metadata := make(map[string]string)
	if dbIntegration.Metadata.Valid {
		var metadataMap map[string]any
		if err := json.Unmarshal(dbIntegration.Metadata.RawMessage, &metadataMap); err != nil {
			return backend.Integration{}, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		for k, v := range metadataMap {
			if str, ok := v.(string); ok {
				metadata[k] = str
			}
		}
	}

	var lastUsedAt *time.Time
	if dbIntegration.LastUsedAt.Valid {
		lastUsedAt = &dbIntegration.LastUsedAt.Time
	}

	return backend.Integration{
		ID:                      dbIntegration.ID,
		OrganizationID:          dbIntegration.OrganizationID,
		UserID:                  dbIntegration.UserID,
		ConnectorType:           backend.ConnectorType(dbIntegration.ConnectorType),
		Status:                  backend.IntegrationStatus(dbIntegration.Status),
		BotID:                   dbIntegration.BotID.String,
		ConnectorUserID:         dbIntegration.ConnectorUserID.String,
		ConnectorOrganizationID: dbIntegration.ConnectorOrganizationID.String,
		Metadata:                metadata,
		CreatedAt:               dbIntegration.CreatedAt,
		UpdatedAt:               dbIntegration.UpdatedAt,
		LastUsedAt:              lastUsedAt,
	}, nil
}
