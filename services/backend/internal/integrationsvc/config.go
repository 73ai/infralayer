package integrationsvc

import (
	"database/sql"
	"fmt"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/connectors/gcp"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/connectors/github"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/connectors/slack"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/supporting/postgres"
)

type Config struct {
	Database *sql.DB       `mapstructure:"-"`
	Slack    slack.Config  `mapstructure:"slack"`
	GitHub   github.Config `mapstructure:"github"`
	GCP      gcp.Config    `mapstructure:"gcp"`
}

func (c Config) New() (backend.IntegrationService, error) {
	integrationRepository := postgres.NewIntegrationRepository(c.Database)

	credentialRepository, err := postgres.NewCredentialRepository(c.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential repository: %w", err)
	}

	connectors := make(map[backend.ConnectorType]domain.Connector)

	if c.Slack.ClientID != "" && c.Slack.BotToken != "" {
		connectors[backend.ConnectorTypeSlack] = c.Slack.New()
	}

	if c.GitHub.AppID != "" {
		c.GitHub.GitHubRepositoryRepo = postgres.NewGitHubRepositoryRepository(c.Database)
		c.GitHub.IntegrationRepository = integrationRepository
		c.GitHub.CredentialRepository = credentialRepository

		connectors[backend.ConnectorTypeGithub] = c.GitHub.New()
	}

	c.GCP.IntegrationRepository = integrationRepository
	c.GCP.CredentialRepository = credentialRepository
	connectors[backend.ConnectorTypeGCP] = c.GCP.New()

	serviceConfig := ServiceConfig{
		IntegrationRepository: integrationRepository,
		CredentialRepository:  credentialRepository,
		Connectors:            connectors,
	}

	return NewService(serviceConfig), nil
}
