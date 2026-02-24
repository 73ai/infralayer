package github

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
	"github.com/golang-jwt/jwt/v4"
)

type Config struct {
	AppID         string `mapstructure:"app_id"`
	AppName       string `mapstructure:"app_name"`
	PrivateKey    string `mapstructure:"private_key"`
	WebhookSecret string `mapstructure:"webhook_secret"`
	RedirectURL   string `mapstructure:"redirect_url"`
	WebhookPort   int    `mapstructure:"webhook_port"`

	GitHubRepositoryRepo  GitHubRepositoryRepository
	IntegrationRepository domain.IntegrationRepository
	CredentialRepository  domain.CredentialRepository
}

func (c Config) New() domain.Connector {
	if c.AppID == "" {
		panic("missing app_id")
	}
	if c.AppName == "" {
		panic("missing app_name")
	}
	if c.PrivateKey == "" {
		panic("missing private_key")
	}
	if c.WebhookSecret == "" {
		panic("missing webhook_secret")
	}
	if c.RedirectURL == "" {
		panic("missing redirect_url")
	}

	var privateKey *rsa.PrivateKey
	var err error
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(c.PrivateKey))
	if err != nil {
		panic(fmt.Sprintf("Failed to parse GitHub private key: %v", err))
	}

	connector := &githubConnector{
		config:     c,
		client:     &http.Client{Timeout: 30 * time.Second},
		privateKey: privateKey,
	}

	return connector
}
