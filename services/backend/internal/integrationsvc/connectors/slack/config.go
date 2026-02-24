package slack

import (
	"net/http"
	"time"

	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
)

type Config struct {
	ClientID      string   `mapstructure:"client_id"`
	ClientSecret  string   `mapstructure:"client_secret"`
	RedirectURL   string   `mapstructure:"redirect_url"`
	Scopes        []string `mapstructure:"-"`
	SigningSecret string   `mapstructure:"signing_secret"`
	BotToken      string   `mapstructure:"bot_token"`
	AppToken      string   `mapstructure:"app_token"`
}

func (c Config) New() domain.Connector {
	c.Scopes = []string{
		"app_mentions:read",
		"chat:write",
		"im:history",
		"im:write",
		"reactions:write",
		"users:read",
		"users:read.email",
		"channels:read",
		"channels:history",
		"groups:read",
		"groups:history",
	}

	if c.RedirectURL == "" {
		panic("missing redirect_url")
	}
	if c.AppToken == "" {
		panic("missing app_token")
	}
	if c.ClientSecret == "" {
		panic("missing client_secret")
	}
	if c.ClientID == "" {
		panic("missing client_id")
	}

	return &slackConnector{
		config: c,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}
