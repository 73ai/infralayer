package integrationapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/generic/httperrors"
	"github.com/google/uuid"
)

type httpHandler struct {
	http.ServeMux
	svc backend.IntegrationService
}

func (h *httpHandler) init() {
	h.HandleFunc("/integrations/initiate/", h.initiate())
	h.HandleFunc("/integrations/authorize/", h.authorize())
	h.HandleFunc("/integrations/sync/", h.sync())
	h.HandleFunc("/integrations/list/", h.list())
	h.HandleFunc("/integrations/revoke/", h.revoke())
	h.HandleFunc("/integrations/status/", h.status())
	h.HandleFunc("/integrations/validate/", h.validateCredentials())
}

func NewHandler(integrationService backend.IntegrationService,
	authMiddleware func(handler http.Handler) http.Handler) http.Handler {
	h := &httpHandler{
		svc: integrationService,
	}

	h.init()
	return authMiddleware(h)
}

func (h *httpHandler) initiate() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		OrganizationID string `json:"organization_id"`
		UserID         string `json:"user_id"`
		ConnectorType  string `json:"connector_type"`
	}
	type response struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		organizationID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			return response{}, fmt.Errorf("invalid organization_id: %w", err)
		}

		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			return response{}, fmt.Errorf("invalid user_id: %w", err)
		}

		cmd := backend.NewIntegrationCommand{
			OrganizationID: organizationID,
			UserID:         userID,
			ConnectorType:  backend.ConnectorType(req.ConnectorType),
		}

		intent, err := h.svc.NewIntegration(ctx, cmd)
		if err != nil {
			return response{}, err
		}

		return response{
			Type: string(intent.Type),
			URL:  intent.URL,
		}, nil
	})
}

func (h *httpHandler) authorize() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		ConnectorType  string `json:"connector_type"`
		Code           string `json:"code"`
		State          string `json:"state"`
		InstallationID string `json:"installation_id"`
	}
	type response struct {
		ID                      string            `json:"id"`
		OrganizationID          string            `json:"organization_id"`
		UserID                  string            `json:"user_id"`
		ConnectorType           string            `json:"connector_type"`
		Status                  string            `json:"status"`
		BotID                   string            `json:"bot_id,omitempty"`
		ConnectorUserID         string            `json:"connector_user_id,omitempty"`
		ConnectorOrganizationID string            `json:"connector_organization_id,omitempty"`
		Metadata                map[string]string `json:"metadata"`
		CreatedAt               string            `json:"created_at"`
		UpdatedAt               string            `json:"updated_at"`
		LastUsedAt              string            `json:"last_used_at,omitempty"`
	}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		cmd := backend.AuthorizeIntegrationCommand{
			ConnectorType:  backend.ConnectorType(req.ConnectorType),
			Code:           req.Code,
			State:          req.State,
			InstallationID: req.InstallationID,
		}

		integration, err := h.svc.AuthorizeIntegration(ctx, cmd)
		if err != nil {
			return response{}, err
		}

		resp := response{
			ID:                      integration.ID.String(),
			OrganizationID:          integration.OrganizationID.String(),
			UserID:                  integration.UserID.String(),
			ConnectorType:           string(integration.ConnectorType),
			Status:                  string(integration.Status),
			BotID:                   integration.BotID,
			ConnectorUserID:         integration.ConnectorUserID,
			ConnectorOrganizationID: integration.ConnectorOrganizationID,
			Metadata:                integration.Metadata,
			CreatedAt:               integration.CreatedAt.Format(time.RFC3339),
			UpdatedAt:               integration.UpdatedAt.Format(time.RFC3339),
		}

		if integration.LastUsedAt != nil {
			resp.LastUsedAt = integration.LastUsedAt.Format(time.RFC3339)
		}

		return resp, nil
	})
}

func (h *httpHandler) list() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		OrganizationID string `json:"organization_id"`
		ConnectorType  string `json:"connector_type,omitempty"`
	}
	type integration struct {
		ID                      string            `json:"id"`
		OrganizationID          string            `json:"organization_id"`
		UserID                  string            `json:"user_id"`
		ConnectorType           string            `json:"connector_type"`
		Status                  string            `json:"status"`
		BotID                   string            `json:"bot_id,omitempty"`
		ConnectorUserID         string            `json:"connector_user_id,omitempty"`
		ConnectorOrganizationID string            `json:"connector_organization_id,omitempty"`
		Metadata                map[string]string `json:"metadata"`
		CreatedAt               string            `json:"created_at"`
		UpdatedAt               string            `json:"updated_at"`
		LastUsedAt              string            `json:"last_used_at,omitempty"`
	}
	type response struct {
		Integrations []integration `json:"integrations"`
	}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		organizationID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			return response{}, fmt.Errorf("invalid organization_id: %w", err)
		}

		query := backend.IntegrationsQuery{
			OrganizationID: organizationID,
		}

		if req.ConnectorType != "" {
			query.ConnectorType = backend.ConnectorType(req.ConnectorType)
		}

		integrations, err := h.svc.Integrations(ctx, query)
		if err != nil {
			return response{}, err
		}

		resp := response{
			Integrations: make([]integration, len(integrations)),
		}

		for i, integ := range integrations {
			resp.Integrations[i] = integration{
				ID:                      integ.ID.String(),
				OrganizationID:          integ.OrganizationID.String(),
				UserID:                  integ.UserID.String(),
				ConnectorType:           string(integ.ConnectorType),
				Status:                  string(integ.Status),
				BotID:                   integ.BotID,
				ConnectorUserID:         integ.ConnectorUserID,
				ConnectorOrganizationID: integ.ConnectorOrganizationID,
				Metadata:                integ.Metadata,
				CreatedAt:               integ.CreatedAt.Format(time.RFC3339),
				UpdatedAt:               integ.UpdatedAt.Format(time.RFC3339),
			}

			if integ.LastUsedAt != nil {
				resp.Integrations[i].LastUsedAt = integ.LastUsedAt.Format(time.RFC3339)
			}
		}

		return resp, nil
	})
}

func (h *httpHandler) revoke() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		IntegrationID  string `json:"integration_id"`
		OrganizationID string `json:"organization_id"`
	}
	type response struct{}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		integrationID, err := uuid.Parse(req.IntegrationID)
		if err != nil {
			return response{}, fmt.Errorf("invalid integration_id: %w", err)
		}

		organizationID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			return response{}, fmt.Errorf("invalid organization_id: %w", err)
		}

		cmd := backend.RevokeIntegrationCommand{
			IntegrationID:  integrationID,
			OrganizationID: organizationID,
		}

		err = h.svc.RevokeIntegration(ctx, cmd)
		return response{}, err
	})
}

func (h *httpHandler) status() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		IntegrationID  string `json:"integration_id"`
		OrganizationID string `json:"organization_id"`
	}
	type response struct {
		ID                      string            `json:"id"`
		OrganizationID          string            `json:"organization_id"`
		UserID                  string            `json:"user_id"`
		ConnectorType           string            `json:"connector_type"`
		Status                  string            `json:"status"`
		BotID                   string            `json:"bot_id,omitempty"`
		ConnectorUserID         string            `json:"connector_user_id,omitempty"`
		ConnectorOrganizationID string            `json:"connector_organization_id,omitempty"`
		Metadata                map[string]string `json:"metadata"`
		CreatedAt               string            `json:"created_at"`
		UpdatedAt               string            `json:"updated_at"`
		LastUsedAt              string            `json:"last_used_at,omitempty"`
		HealthStatus            string            `json:"health_status"`
	}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		integrationID, err := uuid.Parse(req.IntegrationID)
		if err != nil {
			return response{}, fmt.Errorf("invalid integration_id: %w", err)
		}

		organizationID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			return response{}, fmt.Errorf("invalid organization_id: %w", err)
		}

		query := backend.IntegrationQuery{
			IntegrationID:  integrationID,
			OrganizationID: organizationID,
		}

		integration, err := h.svc.Integration(ctx, query)
		if err != nil {
			return response{}, err
		}

		healthStatus := "unknown"

		resp := response{
			ID:                      integration.ID.String(),
			OrganizationID:          integration.OrganizationID.String(),
			UserID:                  integration.UserID.String(),
			ConnectorType:           string(integration.ConnectorType),
			Status:                  string(integration.Status),
			BotID:                   integration.BotID,
			ConnectorUserID:         integration.ConnectorUserID,
			ConnectorOrganizationID: integration.ConnectorOrganizationID,
			Metadata:                integration.Metadata,
			CreatedAt:               integration.CreatedAt.Format(time.RFC3339),
			UpdatedAt:               integration.UpdatedAt.Format(time.RFC3339),
			HealthStatus:            healthStatus,
		}

		if integration.LastUsedAt != nil {
			resp.LastUsedAt = integration.LastUsedAt.Format(time.RFC3339)
		}

		return resp, nil
	})
}

func ApiHandlerFunc[T any, R any](handler func(context.Context, T) (R, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request T
		if r.Method == http.MethodPost && r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		}

		response, err := handler(ctx, request)
		if err != nil {
			slog.Error("error in integration api handler", "path", r.URL, "request", request, "err", err)
			var httpError = httperrors.From(err)
			w.WriteHeader(httpError.HttpStatus)
			_ = json.NewEncoder(w).Encode(httpError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

func (h *httpHandler) sync() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		IntegrationID  string            `json:"integration_id"`
		OrganizationID string            `json:"organization_id"`
		Parameters     map[string]string `json:"parameters,omitempty"`
	}
	type response struct {
		Message string `json:"message"`
	}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		integrationID, err := uuid.Parse(req.IntegrationID)
		if err != nil {
			return response{}, fmt.Errorf("invalid integration_id: %w", err)
		}

		organizationID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			return response{}, fmt.Errorf("invalid organization_id: %w", err)
		}

		cmd := backend.SyncIntegrationCommand{
			IntegrationID:  integrationID,
			OrganizationID: organizationID,
			Parameters:     req.Parameters,
		}

		if cmd.Parameters == nil {
			cmd.Parameters = make(map[string]string)
		}

		if err := h.svc.SyncIntegration(ctx, cmd); err != nil {
			return response{}, fmt.Errorf("failed to sync integration: %w", err)
		}

		return response{
			Message: "Integration synced successfully",
		}, nil
	})
}

func (h *httpHandler) validateCredentials() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		ConnectorType string         `json:"connector_type"`
		Credentials   map[string]any `json:"credentials"`
	}
	type response struct {
		Valid   bool     `json:"valid"`
		Details any      `json:"details,omitempty"`
		Errors  []string `json:"errors,omitempty"`
	}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		if req.ConnectorType == "" {
			return response{
				Valid:  false,
				Errors: []string{"connector_type is required"},
			}, nil
		}

		if req.Credentials == nil || len(req.Credentials) == 0 {
			return response{
				Valid:  false,
				Errors: []string{"credentials are required"},
			}, nil
		}

		validationResult, err := h.svc.ValidateCredentials(ctx, backend.ConnectorType(req.ConnectorType), req.Credentials)
		if err != nil {
			return response{}, fmt.Errorf("failed to validate credentials: %w", err)
		}

		return response{
			Valid:   validationResult.Valid,
			Details: validationResult.Details,
			Errors:  validationResult.Errors,
		}, nil
	})
}
