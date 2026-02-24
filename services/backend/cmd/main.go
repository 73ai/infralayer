package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	agentclient "github.com/73ai/infralayer/services/agent/src/client/go"
	"github.com/73ai/infralayer/services/backend/backendapi"
	"github.com/73ai/infralayer/services/backend/deviceapi"
	"github.com/73ai/infralayer/services/backend/identityapi"
	"github.com/73ai/infralayer/services/backend/integrationapi"
	"github.com/73ai/infralayer/services/backend/internal/conversationsvc"
	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/supporting/agent"
	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/supporting/postgres"
	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/supporting/slack"
	"github.com/73ai/infralayer/services/backend/internal/devicesvc"
	"github.com/73ai/infralayer/services/backend/internal/generic/httplog"
	"github.com/73ai/infralayer/services/backend/internal/generic/postgresconfig"
	"github.com/73ai/infralayer/services/backend/internal/identitysvc"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc"
	"github.com/m-mizutani/masq"
	"golang.org/x/sync/errgroup"

	_ "github.com/lib/pq"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

func main() {
	time.Local = time.UTC

	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	config, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(fmt.Errorf("error reading config file: %w", err))
	}

	var yamlMap map[string]any
	if err := yaml.Unmarshal(config, &yamlMap); err != nil {
		log.Fatalf("Error unmarshalling YAML: %v", err)
	}

	type Config struct {
		LogLevel     string                `mapstructure:"log_level"`
		Port         int                   `mapstructure:"port"`
		GrpcPort     int                   `mapstructure:"grpc_port"`
		HttpLog      bool                  `mapstructure:"http_log"`
		Slack        slack.Config          `mapstructure:"slack"`
		Database     postgresconfig.Config `mapstructure:"database"`
		Agent        agentclient.Config    `mapstructure:"agent"`
		Identity     identitysvc.Config    `mapstructure:"identity"`
		Integrations integrationsvc.Config `mapstructure:"integrations"`
	}

	var c Config
	if err := mapstructure.Decode(yamlMap, &c); err != nil {
		log.Fatalf("Error decoding config: %v", err)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(c.LogLevel)); err != nil {
		panic(err)
	}

	// NOTE: masq library sanitizes sensitive data in logs
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: masq.New(
			masq.WithFieldName("password"),
			masq.WithFieldName("token"),
			masq.WithFieldName("secret"),
			masq.WithFieldName("key"),
			masq.WithFieldName("credential"),
			masq.WithFieldName("auth"),
			masq.WithTag("sensitive"),
		),
	}))
	slog.SetDefault(logger)

	slackConfig := c.Slack
	db, err := postgres.Config{Config: c.Database}.New()
	if err != nil {
		panic(fmt.Errorf("error connecting to database: %w", err))
	}
	slackConfig.WorkSpaceTokenRepository = db
	slackConfig.ChannelRepository = db

	identityService := c.Identity.New(db.DB())
	c.Integrations.Database = db.DB()
	integrationService, err := c.Integrations.New()
	if err != nil {
		panic(fmt.Errorf("error creating integration service: %w", err))
	}

	deviceService := devicesvc.Config{Database: db.DB()}.New()

	authMiddleware := c.Identity.Clerk.NewAuthMiddleware()

	sr, err := slackConfig.New(ctx)
	if err != nil {
		panic(fmt.Errorf("error connecting to slack: %w", err))
	}

	var agentService domain.AgentService
	c.Agent.Timeout = 5 * 60 * time.Second
	c.Agent.ConnectTimeout = 10 * time.Second
	agentClient, err := agent.NewClient(&c.Agent)
	if err != nil {
		log.Printf("Failed to create agent client, falling back to DumbClient: %v", err)
	} else {
		agentService = agentClient
	}

	svcConfig := conversationsvc.Config{
		SlackGateway:           sr,
		IntegrationRepository:  db,
		ConversationRepository: db,
		ChannelRepository:      db,
		AgentService:           agentService,
	}

	svc, err := svcConfig.New(ctx)
	if err != nil {
		panic(fmt.Errorf("error connecting to slack: %w", err))
	}

	g.Go(func() error {
		err = svc.SubscribeSlackNotifications(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			slog.Info("slack notification subscription stopped")
		}
		if err != nil {
			panic(fmt.Errorf("error subscribing to slack notifications: %w", err))
		}
		return nil
	})

	coreAPIHandler := backendapi.NewHandler(svc)
	identityAPIHandler := identityapi.NewHandler(identityService, authMiddleware)
	integrationAPIHandler := integrationapi.NewHandler(integrationService, authMiddleware)
	deviceAPIHandler := deviceapi.NewHandler(deviceService, integrationService, authMiddleware)

	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				slog.Info("backend: http server panic", "recover", r)
			}
		}()
		if strings.HasPrefix(r.URL.Path, "/identity/") {
			identityAPIHandler.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/integrations/") {
			integrationAPIHandler.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/device/") {
			deviceAPIHandler.ServeHTTP(w, r)
			return
		}
		coreAPIHandler.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:        fmt.Sprintf(":%d", c.Port),
		BaseContext: func(net.Listener) context.Context { return ctx },
		Handler:     httplog.Middleware(c.HttpLog)(corsHandler(httpHandler)),
	}

	g.Go(func() error {
		slog.Info("backend: http server starting", "port", c.Port)
		err = httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			slog.Info("backend: http server stopped")
			return nil
		}
		slog.Error("backend: http server failed", "error", err)
		return fmt.Errorf("http server failed: %w", err)
	})

	grpcServer := backendapi.NewGRPCServer(svc)
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", c.GrpcPort))
	if err != nil {
		panic(fmt.Errorf("error creating grpc listener: %w", err))
	}

	g.Go(func() error {
		slog.Info("backend: grpc server starting", "port", c.GrpcPort)
		err = grpcServer.Serve(grpcListener)
		if err != nil {
			slog.Error("backend: grpc server failed", "error", err)
			return fmt.Errorf("grpc server failed: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		slog.Info("backend: identity service webhook server starting", "port", c.Identity.Clerk.Port)
		err = identityService.Subscribe(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			slog.Info("backend: identity service webhook server stopped")
			return nil
		}
		return nil
	})

	g.Go(func() error {
		slog.Info("backend: integration service connectors starting")
		err = integrationService.Subscribe(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			slog.Info("backend: integration service connectors stopped")
			return nil
		}
		slog.Error("backend: integration service connectors failed", "error", err)
		return fmt.Errorf("integration service connectors failed: %w", err)
	})

	if err := g.Wait(); err != nil {
		panic(fmt.Errorf("error waiting for server to finish: %w", err))
	}
}

func corsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}
