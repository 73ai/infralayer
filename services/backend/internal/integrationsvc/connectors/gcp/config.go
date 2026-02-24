package gcp

import (
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
)

// Config holds the configuration for the GCP connector
type Config struct {
	// Repository dependencies
	IntegrationRepository domain.IntegrationRepository `mapstructure:"-"`
	CredentialRepository  domain.CredentialRepository  `mapstructure:"-"`
}

// New creates a new GCP connector instance
func (c Config) New() *Connector {
	return &Connector{
		integrationRepository: c.IntegrationRepository,
		credentialRepository:  c.CredentialRepository,
	}
}
