package clerk

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	clerkapi "github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
)

type clerk struct {
	port          int
	secretKey     string
	webhookSecret string
}

func (c clerk) Subscribe(ctx context.Context, handler func(ctx context.Context, event any) error) error {
	if c.port == 0 {
		panic("clerk: invalid port")
	}
	if c.webhookSecret == "" {
		panic("clerk: webhook secret is empty")
	}
	if c.secretKey == "" {
		panic("clerk: secret key is empty")
	}

	webhookConfig := webhookServerConfig{
		port:                c.port,
		webhookSecret:       c.webhookSecret,
		callbackHandlerFunc: handler,
	}
	return webhookConfig.startWebhookServer(ctx)
}

func (c Config) NewAuthMiddleware() func(http.Handler) http.Handler {
	clerkapi.SetKey(c.SecretKey)

	// Pre-warm JWKS cache
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := jwks.Get(ctx, &jwks.GetParams{}); err != nil {
		slog.Error("failed to pre-warm JWKS cache", "error", err)
	}

	return clerkhttp.WithHeaderAuthorization(
		clerkhttp.Leeway(5 * time.Second),
	)
}
