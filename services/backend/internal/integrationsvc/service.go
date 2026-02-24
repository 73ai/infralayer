package integrationsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/connectors/github"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
	"github.com/google/uuid"
)

type service struct {
	integrationRepository domain.IntegrationRepository
	credentialRepository  domain.CredentialRepository
	connectors            map[backend.ConnectorType]domain.Connector
}

type ServiceConfig struct {
	IntegrationRepository domain.IntegrationRepository
	CredentialRepository  domain.CredentialRepository
	Connectors            map[backend.ConnectorType]domain.Connector
}

func NewService(config ServiceConfig) backend.IntegrationService {
	return &service{
		integrationRepository: config.IntegrationRepository,
		credentialRepository:  config.CredentialRepository,
		connectors:            config.Connectors,
	}
}

func (s *service) NewIntegration(ctx context.Context, cmd backend.NewIntegrationCommand) (backend.IntegrationAuthorizationIntent, error) {
	existingActiveIntegrations, err := s.integrationRepository.FindByOrganizationTypeAndStatus(ctx, cmd.OrganizationID, cmd.ConnectorType, backend.IntegrationStatusActive)
	if err != nil {
		return backend.IntegrationAuthorizationIntent{}, fmt.Errorf("failed to check existing active integrations: %w", err)
	}

	if len(existingActiveIntegrations) > 0 {
		return backend.IntegrationAuthorizationIntent{}, fmt.Errorf("integration already exists for connector type %s", cmd.ConnectorType)
	}

	connector, exists := s.connectors[cmd.ConnectorType]
	if !exists {
		return backend.IntegrationAuthorizationIntent{}, fmt.Errorf("unsupported connector type: %s", cmd.ConnectorType)
	}

	return connector.InitiateAuthorization(cmd.OrganizationID.String(), cmd.UserID.String())
}

func (s *service) AuthorizeIntegration(ctx context.Context, cmd backend.AuthorizeIntegrationCommand) (backend.Integration, error) {
	connector, exists := s.connectors[cmd.ConnectorType]
	if !exists {
		return backend.Integration{}, fmt.Errorf("unsupported connector type: %s", cmd.ConnectorType)
	}

	authData := backend.AuthorizationData{
		Code:           cmd.Code,
		State:          cmd.State,
		InstallationID: cmd.InstallationID,
	}

	credentials, err := connector.CompleteAuthorization(authData)
	if err != nil {
		return backend.Integration{}, fmt.Errorf("failed to complete authorization: %w", err)
	}

	if claimed, exists := credentials.Data["claimed"]; exists && claimed == "true" {
		organizationID, _, err := connector.ParseState(cmd.State)
		if err != nil {
			return backend.Integration{}, fmt.Errorf("failed to parse state: %w", err)
		}

		// Look for active integrations only - inactive ones should be handled by ClaimInstallation
		existingActiveIntegrations, err := s.integrationRepository.FindByOrganizationTypeAndStatus(ctx, organizationID, cmd.ConnectorType, backend.IntegrationStatusActive)
		if err != nil {
			return backend.Integration{}, fmt.Errorf("failed to find claimed integration: %w", err)
		}

		for _, integration := range existingActiveIntegrations {
			if integration.BotID == cmd.InstallationID {
				return integration, nil
			}
		}

		return backend.Integration{}, fmt.Errorf("claimed integration not found")
	}

	organizationID, userID, err := connector.ParseState(cmd.State)
	if err != nil {
		return backend.Integration{}, fmt.Errorf("failed to parse state: %w", err)
	}

	existingActiveIntegrations, err := s.integrationRepository.FindByOrganizationTypeAndStatus(ctx, organizationID, cmd.ConnectorType, backend.IntegrationStatusActive)
	if err != nil {
		return backend.Integration{}, fmt.Errorf("failed to check existing active integrations: %w", err)
	}

	if len(existingActiveIntegrations) > 0 {
		return backend.Integration{}, fmt.Errorf("integration already exists for connector type %s in organization %s", cmd.ConnectorType, organizationID)
	}

	now := time.Now()
	integration := backend.Integration{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		UserID:         userID,
		ConnectorType:  cmd.ConnectorType,
		Status:         backend.IntegrationStatusActive,
		Metadata:       make(map[string]string),
		CreatedAt:      now,
		UpdatedAt:      now,
		LastUsedAt:     &now,
	}

	if cmd.InstallationID != "" {
		integration.BotID = cmd.InstallationID
	}

	if credentials.OrganizationInfo != nil {
		integration.ConnectorOrganizationID = credentials.OrganizationInfo.ExternalID
		integration.Metadata["connector_org_name"] = credentials.OrganizationInfo.Name
		for k, v := range credentials.OrganizationInfo.Metadata {
			integration.Metadata[k] = v
		}
	}

	if err := s.integrationRepository.Store(ctx, integration); err != nil {
		return backend.Integration{}, fmt.Errorf("failed to store integration: %w", err)
	}

	credentialRecord := domain.IntegrationCredential{
		ID:              uuid.New(),
		IntegrationID:   integration.ID,
		CredentialType:  credentials.Type,
		Data:            credentials.Data,
		ExpiresAt:       credentials.ExpiresAt,
		EncryptionKeyID: "v1",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.credentialRepository.Store(ctx, credentialRecord); err != nil {
		return backend.Integration{}, fmt.Errorf("failed to store credentials: %w", err)
	}

	return integration, nil
}

func (s *service) RevokeIntegration(ctx context.Context, cmd backend.RevokeIntegrationCommand) error {
	integration, err := s.integrationRepository.FindByID(ctx, cmd.IntegrationID)
	if err != nil {
		return fmt.Errorf("failed to find integration: %w", err)
	}

	if integration.OrganizationID != cmd.OrganizationID {
		return fmt.Errorf("integration not found for organization")
	}

	credential, err := s.credentialRepository.FindByIntegration(ctx, cmd.IntegrationID)
	if err != nil {
		return fmt.Errorf("failed to find credentials: %w", err)
	}

	connector, exists := s.connectors[integration.ConnectorType]
	if exists {
		creds := backend.Credentials{
			Type:      credential.CredentialType,
			Data:      credential.Data,
			ExpiresAt: credential.ExpiresAt,
		}

		if err := connector.RevokeCredentials(creds); err != nil {
			return fmt.Errorf("failed to revoke credentials with connector: %w", err)
		}
	}

	if err := s.credentialRepository.Delete(ctx, cmd.IntegrationID); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	if err := s.integrationRepository.Delete(ctx, cmd.IntegrationID); err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	return nil
}

func (s *service) Integrations(ctx context.Context, query backend.IntegrationsQuery) ([]backend.Integration, error) {
	if query.ConnectorType != "" && query.Status != "" {
		return s.integrationRepository.FindByOrganizationTypeAndStatus(ctx, query.OrganizationID, query.ConnectorType, query.Status)
	}

	if query.ConnectorType != "" {
		return s.integrationRepository.FindByOrganizationAndType(ctx, query.OrganizationID, query.ConnectorType)
	}

	if query.Status != "" {
		return s.integrationRepository.FindByOrganizationAndStatus(ctx, query.OrganizationID, query.Status)
	}

	return s.integrationRepository.FindByOrganization(ctx, query.OrganizationID)
}

func (s *service) Integration(ctx context.Context, query backend.IntegrationQuery) (backend.Integration, error) {
	integration, err := s.integrationRepository.FindByID(ctx, query.IntegrationID)
	if err != nil {
		return backend.Integration{}, fmt.Errorf("failed to find integration: %w", err)
	}

	if integration.OrganizationID != query.OrganizationID {
		return backend.Integration{}, fmt.Errorf("integration not found for organization")
	}

	return integration, nil
}

func (s *service) IntegrationCredentials(ctx context.Context, query backend.IntegrationCredentialsQuery) (backend.Credentials, error) {
	integration, err := s.integrationRepository.FindByID(ctx, query.IntegrationID)
	if err != nil {
		return backend.Credentials{}, fmt.Errorf("failed to find integration: %w", err)
	}

	if integration.OrganizationID != query.OrganizationID {
		return backend.Credentials{}, fmt.Errorf("integration not found for organization")
	}

	credential, err := s.credentialRepository.FindByIntegration(ctx, query.IntegrationID)
	if err != nil {
		return backend.Credentials{}, fmt.Errorf("failed to find credentials: %w", err)
	}

	return backend.Credentials{
		Type:      credential.CredentialType,
		Data:      credential.Data,
		ExpiresAt: credential.ExpiresAt,
	}, nil
}

func (s *service) SyncIntegration(ctx context.Context, cmd backend.SyncIntegrationCommand) error {
	integration, err := s.integrationRepository.FindByID(ctx, cmd.IntegrationID)
	if err != nil {
		return fmt.Errorf("failed to find integration: %w", err)
	}

	if integration.OrganizationID != cmd.OrganizationID {
		return fmt.Errorf("integration not found for organization")
	}

	connector, exists := s.connectors[integration.ConnectorType]
	if !exists {
		return fmt.Errorf("unsupported connector type: %s", integration.ConnectorType)
	}

	if err := connector.Sync(ctx, integration, cmd.Parameters); err != nil {
		return fmt.Errorf("failed to sync integration: %w", err)
	}

	now := time.Now()
	integration.LastUsedAt = &now
	integration.UpdatedAt = now

	if err := s.integrationRepository.Update(ctx, integration); err != nil {
		return fmt.Errorf("failed to update integration: %w", err)
	}

	return nil
}

func (s *service) ValidateCredentials(ctx context.Context, connectorType backend.ConnectorType, credentials map[string]any) (backend.CredentialValidationResult, error) {
	connector, exists := s.connectors[connectorType]
	if !exists {
		return backend.CredentialValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Unsupported connector type: %s", connectorType)},
		}, nil
	}

	credData := make(map[string]string)

	switch connectorType {
	case backend.ConnectorTypeGCP:
		if saJSON, ok := credentials["service_account_json"].(string); ok {
			credData["service_account_json"] = saJSON
		} else {
			return backend.CredentialValidationResult{
				Valid:  false,
				Errors: []string{"service_account_json is required for GCP connector"},
			}, nil
		}
	default:
		for k, v := range credentials {
			if str, ok := v.(string); ok {
				credData[k] = str
			} else {
				credData[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	creds := backend.Credentials{
		Type: backend.CredentialTypeServiceAccount, // Default, connectors can override
		Data: credData,
	}

	// Validate using the connector
	err := connector.ValidateCredentials(creds)
	if err != nil {
		return backend.CredentialValidationResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}, nil
	}

	return backend.CredentialValidationResult{
		Valid:  true,
		Errors: []string{},
	}, nil
}

// Subscribe starts webhook subscriptions for all connectors
func (s *service) Subscribe(ctx context.Context) error {
	for connectorType, connector := range s.connectors {
		go func(connectorType backend.ConnectorType, connector domain.Connector) {
			if err := connector.Subscribe(ctx, s.handleConnectorEvent); err != nil {
				slog.Error("connector subscription failed", "connector_type", connectorType, "error", err)
			}
		}(connectorType, connector)
	}

	return nil
}

func (s *service) handleConnectorEvent(ctx context.Context, event any) error {
	switch e := event.(type) {
	case github.WebhookEvent:
		if connector, exists := s.connectors[backend.ConnectorTypeGithub]; exists {
			return connector.ProcessEvent(ctx, e)
		}
		return fmt.Errorf("GitHub connector not found")
	default:
		slog.Debug("received unknown event type", "event_type", fmt.Sprintf("%T", event))
		return nil
	}
}
