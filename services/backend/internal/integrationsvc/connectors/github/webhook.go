package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
	"github.com/google/uuid"
)

func (g *githubConnector) ValidateWebhookSignature(payload []byte, signature string, secret string) error {
	if secret == "" {
		secret = g.config.WebhookSecret
	}

	expectedSignature := g.computeSignature(payload, secret)

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("webhook signature validation failed")
	}

	return nil
}

func (g *githubConnector) computeSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return fmt.Sprintf("sha256=%s", hex.EncodeToString(h.Sum(nil)))
}

func (g *githubConnector) ProcessEvent(ctx context.Context, event any) error {
	webhookEvent, ok := event.(WebhookEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected WebhookEvent")
	}

	switch webhookEvent.EventType {
	case EventTypeInstallation:
		return g.handleInstallationEvent(ctx, webhookEvent)
	case "installation_repositories":
		return g.handleInstallationRepositoriesEvent(ctx, webhookEvent)
	default:
		slog.Debug("ignoring non-installation event",
			"event_type", webhookEvent.EventType,
			"installation_id", webhookEvent.InstallationID)
		return nil
	}
}

func (g *githubConnector) Subscribe(ctx context.Context, handler func(ctx context.Context, event any) error) error {
	if g.config.WebhookPort == 0 {
		return fmt.Errorf("github: webhook port is required for webhook server")
	}

	webhookConfig := webhookServerConfig{
		port:                g.config.WebhookPort,
		webhookSecret:       g.config.WebhookSecret,
		callbackHandlerFunc: handler,
		validateSignature:   g.ValidateWebhookSignature,
	}

	return webhookConfig.startWebhookServer(ctx)
}

func (g *githubConnector) handleInstallationEvent(ctx context.Context, event WebhookEvent) error {
	slog.Info("handling GitHub installation event",
		"action", event.InstallationAction,
		"installation_id", event.InstallationID,
		"account_login", event.SenderLogin,
		"repositories_added", len(event.RepositoriesAdded),
		"repositories_removed", len(event.RepositoriesRemoved))

	installationEvent, err := g.parseInstallationEvent(event.RawPayload)
	if err != nil {
		return fmt.Errorf("failed to parse installation event: %w", err)
	}
	switch installationEvent.Action {
	case "created":
		return g.handleInstallationCreated(ctx, installationEvent)
	case "deleted":
		return g.handleInstallationDeleted(ctx, installationEvent)
	case "suspend":
		return g.handleInstallationSuspended(ctx, installationEvent)
	case "unsuspend":
		return g.handleInstallationUnsuspended(ctx, installationEvent)
	case "new_permissions_accepted":
		return g.handlePermissionsUpdated(ctx, installationEvent)
	default:
		slog.Debug("unhandled installation action", "action", installationEvent.Action)
		return nil
	}
}

func (g *githubConnector) parseInstallationEvent(rawPayload map[string]any) (InstallationEvent, error) {
	payloadBytes, err := json.Marshal(rawPayload)
	if err != nil {
		return InstallationEvent{}, fmt.Errorf("failed to marshal raw payload: %w", err)
	}

	var installationEvent InstallationEvent
	if err := json.Unmarshal(payloadBytes, &installationEvent); err != nil {
		return InstallationEvent{}, fmt.Errorf("failed to unmarshal installation event: %w", err)
	}

	installationEvent.RawPayload = rawPayload
	return installationEvent, nil
}

func (g *githubConnector) handleInstallationCreated(ctx context.Context, event InstallationEvent) error {
	slog.Info("GitHub App installation created",
		"installation_id", event.Installation.ID,
		"account", event.Installation.Account.Login,
		"repository_selection", event.Installation.RepositorySelection,
		"repository_count", len(event.Repositories))

	slog.Info("GitHub installation created - will be claimed during authorization flow",
		"installation_id", event.Installation.ID,
		"account_login", event.Installation.Account.Login,
		"account_type", event.Installation.Account.Type)

	return nil
}

func (g *githubConnector) handleInstallationDeleted(ctx context.Context, event InstallationEvent) error {
	slog.Info("GitHub App installation deleted",
		"installation_id", event.Installation.ID,
		"account", event.Installation.Account.Login)

	installationIDStr := strconv.FormatInt(event.Installation.ID, 10)
	integration, err := g.config.IntegrationRepository.FindByBotIDAndType(ctx, installationIDStr, backend.ConnectorTypeGithub)
	if err != nil {
		if errors.Is(err, domain.ErrIntegrationNotFound) {
			slog.Debug("integration not found for deleted installation",
				"installation_id", event.Installation.ID)
			return nil
		}
		return fmt.Errorf("failed to find integration for deleted installation %d: %w", event.Installation.ID, err)
	}
	if err := g.config.IntegrationRepository.UpdateStatus(ctx, integration.ID, backend.IntegrationStatusInactive); err != nil {
		return fmt.Errorf("failed to update integration status for deleted installation %d: %w", event.Installation.ID, err)
	}

	slog.Info("GitHub integration marked as inactive due to installation deletion",
		"installation_id", event.Installation.ID,
		"integration_id", integration.ID,
		"organization_id", integration.OrganizationID)

	return nil
}

func (g *githubConnector) handleInstallationSuspended(ctx context.Context, event InstallationEvent) error {
	slog.Info("GitHub App installation suspended",
		"installation_id", event.Installation.ID,
		"account", event.Installation.Account.Login,
		"suspended_by", event.Installation.SuspendedBy)

	installationIDStr := strconv.FormatInt(event.Installation.ID, 10)
	integration, err := g.config.IntegrationRepository.FindByBotIDAndType(ctx, installationIDStr, backend.ConnectorTypeGithub)
	if err != nil {
		if errors.Is(err, domain.ErrIntegrationNotFound) {
			slog.Debug("integration not found for suspended installation",
				"installation_id", event.Installation.ID)
			return nil
		}
		return fmt.Errorf("failed to find integration for suspended installation %d: %w", event.Installation.ID, err)
	}
	if err := g.config.IntegrationRepository.UpdateStatus(ctx, integration.ID, backend.IntegrationStatusSuspended); err != nil {
		return fmt.Errorf("failed to update integration status to suspended for installation %d: %w", event.Installation.ID, err)
	}

	slog.Info("GitHub integration status updated to suspended",
		"installation_id", event.Installation.ID,
		"integration_id", integration.ID,
		"organization_id", integration.OrganizationID)

	return nil
}

func (g *githubConnector) handleInstallationUnsuspended(ctx context.Context, event InstallationEvent) error {
	slog.Info("GitHub App installation unsuspended",
		"installation_id", event.Installation.ID,
		"account", event.Installation.Account.Login)

	installationIDStr := strconv.FormatInt(event.Installation.ID, 10)
	integration, err := g.config.IntegrationRepository.FindByBotIDAndType(ctx, installationIDStr, backend.ConnectorTypeGithub)
	if err != nil {
		if errors.Is(err, domain.ErrIntegrationNotFound) {
			slog.Debug("integration not found for unsuspended installation",
				"installation_id", event.Installation.ID)
			return nil
		}
		return fmt.Errorf("failed to find integration for unsuspended installation %d: %w", event.Installation.ID, err)
	}

	if err := g.config.IntegrationRepository.UpdateStatus(ctx, integration.ID, backend.IntegrationStatusActive); err != nil {
		return fmt.Errorf("failed to update integration status to active for installation %d: %w", event.Installation.ID, err)
	}

	slog.Info("GitHub integration status updated to active",
		"installation_id", event.Installation.ID,
		"integration_id", integration.ID,
		"organization_id", integration.OrganizationID)

	return nil
}

func (g *githubConnector) handlePermissionsUpdated(ctx context.Context, event InstallationEvent) error {
	slog.Info("GitHub App permissions updated",
		"installation_id", event.Installation.ID,
		"account", event.Installation.Account.Login,
		"permissions", event.Installation.Permissions)

	isSuspended, err := g.isInstallationSuspended(ctx, event.Installation.ID)
	if err != nil {
		return fmt.Errorf("failed to check installation suspension status: %w", err)
	}

	if isSuspended {
		slog.Info("skipping permissions update processing for suspended installation",
			"installation_id", event.Installation.ID,
			"account", event.Installation.Account.Login)
		return nil
	}

	installationIDStr := strconv.FormatInt(event.Installation.ID, 10)
	integration, err := g.config.IntegrationRepository.FindByBotIDAndType(ctx, installationIDStr, backend.ConnectorTypeGithub)
	if err != nil {
		if errors.Is(err, domain.ErrIntegrationNotFound) {
			slog.Debug("integration not found for permissions update",
				"installation_id", event.Installation.ID)
			return nil
		}
		return fmt.Errorf("failed to find integration for permissions update %d: %w", event.Installation.ID, err)
	}

	updatedMetadata := make(map[string]string)
	for k, v := range integration.Metadata {
		updatedMetadata[k] = v
	}

	for permission, access := range event.Installation.Permissions {
		updatedMetadata["permission_"+permission] = access
	}
	if err := g.config.IntegrationRepository.UpdateMetadata(ctx, integration.ID, updatedMetadata); err != nil {
		return fmt.Errorf("failed to update integration metadata for installation %d: %w", event.Installation.ID, err)
	}

	slog.Info("GitHub integration permissions updated successfully",
		"installation_id", event.Installation.ID,
		"integration_id", integration.ID,
		"organization_id", integration.OrganizationID,
		"permissions_count", len(event.Installation.Permissions))

	if integration.Status == backend.IntegrationStatusActive {
		integrationUUID := integration.ID

		slog.Info("syncing repositories after permissions update",
			"installation_id", event.Installation.ID,
			"integration_id", integration.ID)

		if err := g.syncRepositories(ctx, integrationUUID, installationIDStr); err != nil {
			slog.Error("failed to sync repositories after permissions update",
				"installation_id", event.Installation.ID,
				"integration_id", integration.ID,
				"error", err)
		} else {
			slog.Info("repository sync completed successfully after permissions update",
				"installation_id", event.Installation.ID,
				"integration_id", integration.ID)
		}
	} else {
		slog.Info("skipping repository sync for inactive integration",
			"installation_id", event.Installation.ID,
			"integration_id", integration.ID,
			"status", integration.Status)
	}

	return nil
}

func (g *githubConnector) handleInstallationRepositoriesEvent(ctx context.Context, event WebhookEvent) error {
	slog.Info("handling GitHub installation repositories event",
		"action", event.Action,
		"installation_id", event.InstallationID,
		"repositories_added", len(event.RepositoriesAdded),
		"repositories_removed", len(event.RepositoriesRemoved))

	installationEvent, err := g.parseInstallationEvent(event.RawPayload)
	if err != nil {
		return fmt.Errorf("failed to parse installation repositories event: %w", err)
	}
	switch installationEvent.Action {
	case "added":
		return g.handleRepositoriesAdded(ctx, installationEvent)
	case "removed":
		return g.handleRepositoriesRemoved(ctx, installationEvent)
	default:
		slog.Debug("unhandled installation repositories action", "action", installationEvent.Action)
		return nil
	}
}

func (g *githubConnector) handleRepositoriesAdded(ctx context.Context, event InstallationEvent) error {
	slog.Info("GitHub App repositories added",
		"installation_id", event.Installation.ID,
		"account", event.Installation.Account.Login,
		"repositories_count", len(event.RepositoriesAdded))

	installationIDStr := strconv.FormatInt(event.Installation.ID, 10)
	isSuspended, err := g.isInstallationSuspended(ctx, event.Installation.ID)
	if err != nil {
		return fmt.Errorf("failed to check installation suspension status: %w", err)
	}

	if isSuspended {
		slog.Info("skipping repository addition processing for suspended installation",
			"installation_id", event.Installation.ID,
			"account", event.Installation.Account.Login,
			"repositories_count", len(event.RepositoriesAdded))
		return nil
	}

	integrationID, err := g.findIntegrationIDByInstallationID(ctx, installationIDStr)
	if err != nil {
		slog.Error("failed to find integration for repository addition",
			"installation_id", event.Installation.ID,
			"error", err)
		return nil
	}

	if integrationID != uuid.Nil {
		if err := g.addRepositories(ctx, integrationID, event.RepositoriesAdded); err != nil {
			return fmt.Errorf("failed to add repositories: %w", err)
		}
	}

	return nil
}

func (g *githubConnector) handleRepositoriesRemoved(ctx context.Context, event InstallationEvent) error {
	slog.Info("GitHub App repositories removed",
		"installation_id", event.Installation.ID,
		"account", event.Installation.Account.Login,
		"repositories_count", len(event.RepositoriesRemoved))

	installationIDStr := strconv.FormatInt(event.Installation.ID, 10)
	isSuspended, err := g.isInstallationSuspended(ctx, event.Installation.ID)
	if err != nil {
		return fmt.Errorf("failed to check installation suspension status: %w", err)
	}

	if isSuspended {
		slog.Info("skipping repository removal processing for suspended installation",
			"installation_id", event.Installation.ID,
			"account", event.Installation.Account.Login,
			"repositories_count", len(event.RepositoriesRemoved))
		return nil
	}

	integrationID, err := g.findIntegrationIDByInstallationID(ctx, installationIDStr)
	if err != nil {
		slog.Error("failed to find integration for repository removal",
			"installation_id", event.Installation.ID,
			"error", err)
		return nil
	}

	if integrationID != uuid.Nil {
		var repoIDs []int64
		for _, repo := range event.RepositoriesRemoved {
			repoIDs = append(repoIDs, repo.ID)
		}
		if err := g.removeRepositories(ctx, integrationID, repoIDs); err != nil {
			return fmt.Errorf("failed to remove repositories: %w", err)
		}
	}

	return nil
}

func (g *githubConnector) isInstallationSuspended(ctx context.Context, installationID int64) (bool, error) {
	installationIDStr := strconv.FormatInt(installationID, 10)
	integration, err := g.config.IntegrationRepository.FindByBotIDAndType(ctx, installationIDStr, backend.ConnectorTypeGithub)
	if err != nil {
		if errors.Is(err, domain.ErrIntegrationNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to find integration by installation ID %d: %w", installationID, err)
	}

	return integration.Status == backend.IntegrationStatusSuspended, nil
}

func (g *githubConnector) findIntegrationIDByInstallationID(ctx context.Context, installationID string) (uuid.UUID, error) {
	integration, err := g.config.IntegrationRepository.FindByBotIDAndType(ctx, installationID, backend.ConnectorTypeGithub)
	if err != nil {
		if errors.Is(err, domain.ErrIntegrationNotFound) {
			slog.Debug("integration not found for installation ID", "installation_id", installationID)
			return uuid.Nil, nil
		}
		return uuid.Nil, fmt.Errorf("failed to find integration by installation ID: %w", err)
	}

	integrationUUID := integration.ID

	slog.Debug("found integration for installation ID",
		"installation_id", installationID,
		"integration_id", integration.ID,
		"organization_id", integration.OrganizationID)

	return integrationUUID, nil
}

func convertUserToMap(user *User) map[string]any {
	if user == nil {
		return nil
	}
	return map[string]any{
		"id":    user.ID,
		"login": user.Login,
		"type":  user.Type,
	}
}

// Webhook server configuration and implementation
type webhookServerConfig struct {
	port                int
	webhookSecret       string
	callbackHandlerFunc func(ctx context.Context, event any) error
	validateSignature   func(payload []byte, signature string, secret string) error
}

func (c webhookServerConfig) startWebhookServer(ctx context.Context) error {
	h := &webhookHandler{
		callbackHandlerFunc: c.callbackHandlerFunc,
	}
	h.init()

	httpServer := &http.Server{
		Addr:        fmt.Sprintf(":%d", c.port),
		BaseContext: func(net.Listener) context.Context { return ctx },
		Handler:     panicMiddleware(webhookValidationMiddleware(c.webhookSecret, c.validateSignature, h)),
	}

	return httpServer.ListenAndServe()
}

type webhookHandler struct {
	http.ServeMux
	callbackHandlerFunc func(ctx context.Context, event any) error
}

func (wh *webhookHandler) init() {
	wh.HandleFunc("/webhooks/github", wh.handler())
}

func (wh *webhookHandler) handler() func(w http.ResponseWriter, r *http.Request) {
	type response struct{}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventType := r.Header.Get("X-GitHub-Event")
		if eventType == "" {
			http.Error(w, "Missing X-GitHub-Event header", http.StatusBadRequest)
			return
		}

		if eventType != "installation" && eventType != "installation_repositories" {
			slog.Debug("ignoring non-installation event", "event_type", eventType)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response{})
			return
		}

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read payload", http.StatusBadRequest)
			return
		}

		var rawPayload map[string]any
		if err := json.Unmarshal(payload, &rawPayload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		webhookEvent, err := wh.convertToWebhookEvent(eventType, rawPayload)
		if err != nil {
			slog.Error("failed to convert GitHub webhook event", "event_type", eventType, "error", err)
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

		if err := wh.callbackHandlerFunc(ctx, webhookEvent); err != nil {
			slog.Error("error handling GitHub webhook event", "event_type", eventType, "error", err)
			http.Error(w, "Failed to handle event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response{})
	}
}

func (wh *webhookHandler) convertToWebhookEvent(eventType string, rawPayload map[string]any) (WebhookEvent, error) {
	event := WebhookEvent{
		EventType:  EventType(eventType),
		RawPayload: rawPayload,
		CreatedAt:  time.Now(),
	}

	if installation, ok := rawPayload["installation"].(map[string]any); ok {
		if id, ok := installation["id"].(float64); ok {
			event.InstallationID = strconv.FormatFloat(id, 'f', 0, 64)
		}
	}

	if sender, ok := rawPayload["sender"].(map[string]any); ok {
		if id, ok := sender["id"].(float64); ok {
			event.SenderID = int64(id)
		}
		if login, ok := sender["login"].(string); ok {
			event.SenderLogin = login
		}
	}

	if action, ok := rawPayload["action"].(string); ok {
		event.Action = action
		event.InstallationAction = action
	}

	if eventType == "installation" || eventType == "installation_repositories" {
		if repositories, ok := rawPayload["repositories"].([]any); ok {
			for _, repo := range repositories {
				if repoMap, ok := repo.(map[string]any); ok {
					if fullName, ok := repoMap["full_name"].(string); ok {
						event.RepositoriesAdded = append(event.RepositoriesAdded, fullName)
					}
				}
			}
		}

		if repositoriesRemoved, ok := rawPayload["repositories_removed"].([]any); ok {
			for _, repo := range repositoriesRemoved {
				if repoMap, ok := repo.(map[string]any); ok {
					if fullName, ok := repoMap["full_name"].(string); ok {
						event.RepositoriesRemoved = append(event.RepositoriesRemoved, fullName)
					}
				}
			}
		}
	}

	return event, nil
}

func panicMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("github: panic while handling http request", "recover", r)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func webhookValidationMiddleware(webhookSecret string, validateSignature func(payload []byte, signature string, secret string) error, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if webhookSecret == "" {
			next.ServeHTTP(w, r)
			return
		}

		signature := r.Header.Get("X-Hub-Signature-256")
		if signature == "" {
			slog.Info("github: missing webhook signature")
			http.Error(w, "Missing webhook signature", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		if err := validateSignature(body, signature, webhookSecret); err != nil {
			slog.Info("github: webhook validation failed", "signature", signature, "error", err)
			http.Error(w, "Invalid webhook signature", http.StatusUnauthorized)
			return
		}

		r.Body = io.NopCloser(strings.NewReader(string(body)))
		next.ServeHTTP(w, r)
	})
}

func timeValueFromPointer(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
