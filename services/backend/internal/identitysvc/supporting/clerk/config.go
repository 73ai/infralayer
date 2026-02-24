package clerk

import "github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"

type Config struct {
	Port          int    `mapstructure:"port"`
	WebhookSecret string `mapstructure:"webhook_secret"`
	SecretKey     string `mapstructure:"secret_key"`
}

func (c Config) NewAuthService() domain.AuthService {
	return &clerk{
		port:          c.Port,
		secretKey:     c.SecretKey,
		webhookSecret: c.WebhookSecret,
	}
}
