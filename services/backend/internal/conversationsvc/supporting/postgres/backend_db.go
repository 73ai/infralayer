package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
	"github.com/google/uuid"
)

type BackendDB struct {
	db *sql.DB
	Querier
}

func (i *BackendDB) DB() *sql.DB {
	return i.db
}

var _ domain.WorkSpaceTokenRepository = (*BackendDB)(nil)
var _ domain.IntegrationRepository = (*BackendDB)(nil)
var _ domain.ConversationRepository = (*BackendDB)(nil)
var _ domain.ChannelRepository = (*BackendDB)(nil)

func (i BackendDB) SaveToken(ctx context.Context, teamID, token string) error {
	err := i.saveSlackToken(ctx, saveSlackTokenParams{
		TeamID:  teamID,
		Token:   token,
		TokenID: uuid.New(),
	})
	if err != nil {
		return fmt.Errorf("failed to save slack token: %w", err)
	}
	return nil
}

func (i BackendDB) GetToken(ctx context.Context, teamID string) (string, error) {
	token, err := i.slackToken(ctx, teamID)
	if err != nil {
		return "", fmt.Errorf("failed to get slack token: %w", err)
	}
	return token, nil
}

func (i BackendDB) Integrations(ctx context.Context, businessID uuid.UUID) ([]domain.Integration, error) {
	is, err := i.integrations(ctx, businessID)
	if err != nil {
		return nil, err
	}

	var integrations []domain.Integration
	for _, i := range is {
		integrations = append(integrations, domain.Integration{
			Integration: backend.Integration{
				ConnectorType: backend.ConnectorType(i.Provider),
				Status:        backend.IntegrationStatus(i.Status),
			},
			BusinessID:        businessID.String(),
			ProviderProjectID: i.ProviderProjectID,
		})
	}

	return integrations, nil
}

func (i BackendDB) SaveIntegration(ctx context.Context, integration domain.Integration) error {
	bid := uuid.MustParse(integration.BusinessID)
	err := i.saveIntegration(ctx, saveIntegrationParams{
		ID:                uuid.New(),
		BusinessID:        bid,
		Provider:          string(integration.ConnectorType),
		ProviderProjectID: integration.ProviderProjectID,
		Status:            string(integration.Status),
	})
	if err != nil {
		return fmt.Errorf("failed to save integration: %w", err)
	}
	return nil
}
