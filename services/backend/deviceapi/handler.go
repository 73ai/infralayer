package deviceapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/devicesvc"
	"github.com/73ai/infralayer/services/backend/internal/devicesvc/domain"
	"github.com/73ai/infralayer/services/backend/internal/generic/httperrors"
	"github.com/google/uuid"
)

type httpHandler struct {
	http.ServeMux
	svc                 *devicesvc.Service
	integrationService  backend.IntegrationService
	clerkAuthMiddleware func(http.Handler) http.Handler
}

func (h *httpHandler) init() {
	// Public endpoints (no auth)
	h.HandleFunc("/device/auth/initiate", h.initiateDeviceFlow())
	h.HandleFunc("/device/auth/poll", h.pollDeviceFlow())

	// Clerk-protected endpoint (for web app)
	h.Handle("/device/auth/authorize", h.clerkAuthMiddleware(http.HandlerFunc(h.authorizeDevice())))

	// Device token-protected endpoints
	h.HandleFunc("/device/auth/refresh", h.refreshToken())
	h.HandleFunc("/device/auth/revoke", h.revokeToken())
	h.HandleFunc("/device/credentials/gcp", h.getGCPCredentials())
	h.HandleFunc("/device/credentials/gke", h.getGKEClusterInfo())
}

func NewHandler(
	deviceService *devicesvc.Service,
	integrationService backend.IntegrationService,
	clerkAuthMiddleware func(http.Handler) http.Handler,
) http.Handler {
	h := &httpHandler{
		svc:                 deviceService,
		integrationService:  integrationService,
		clerkAuthMiddleware: clerkAuthMiddleware,
	}
	h.init()
	return h
}

func (h *httpHandler) initiateDeviceFlow() http.HandlerFunc {
	type response struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURL string `json:"verification_url"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		result, err := h.svc.InitiateDeviceFlow(r.Context())
		if err != nil {
			slog.Error("failed to initiate device flow", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp := response{
			DeviceCode:      result.DeviceCode,
			UserCode:        result.UserCode,
			VerificationURL: result.VerificationURL,
			ExpiresIn:       result.ExpiresIn,
			Interval:        result.Interval,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (h *httpHandler) pollDeviceFlow() http.HandlerFunc {
	type request struct {
		DeviceCode string `json:"device_code"`
	}
	type response struct {
		Authorized     bool   `json:"authorized"`
		AccessToken    string `json:"access_token,omitempty"`
		RefreshToken   string `json:"refresh_token,omitempty"`
		ExpiresIn      int    `json:"expires_in,omitempty"`
		OrganizationID string `json:"organization_id,omitempty"`
		UserID         string `json:"user_id,omitempty"`
		Error          string `json:"error,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		result, err := h.svc.PollDeviceFlow(r.Context(), req.DeviceCode)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")

			if errors.Is(err, domain.ErrAuthorizationPending) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(response{
					Authorized: false,
					Error:      "authorization_pending",
				})
				return
			}

			if errors.Is(err, domain.ErrDeviceCodeExpired) {
				w.WriteHeader(http.StatusGone)
				_ = json.NewEncoder(w).Encode(response{
					Authorized: false,
					Error:      "expired_token",
				})
				return
			}

			if errors.Is(err, domain.ErrDeviceCodeNotFound) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(response{
					Authorized: false,
					Error:      "invalid_device_code",
				})
				return
			}

			slog.Error("failed to poll device flow", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp := response{
			Authorized:     result.Authorized,
			AccessToken:    result.AccessToken,
			RefreshToken:   result.RefreshToken,
			ExpiresIn:      result.ExpiresIn,
			OrganizationID: result.OrganizationID.String(),
			UserID:         result.UserID.String(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (h *httpHandler) authorizeDevice() http.HandlerFunc {
	type request struct {
		UserCode       string `json:"user_code"`
		OrganizationID string `json:"organization_id"`
		UserID         string `json:"user_id"`
	}
	type response struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		orgID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(response{Success: false, Error: "invalid organization_id"})
			return
		}

		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(response{Success: false, Error: "invalid user_id"})
			return
		}

		err = h.svc.AuthorizeDevice(r.Context(), req.UserCode, orgID, userID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")

			if errors.Is(err, domain.ErrDeviceCodeNotFound) || errors.Is(err, domain.ErrInvalidUserCode) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(response{Success: false, Error: "invalid_user_code"})
				return
			}

			if errors.Is(err, domain.ErrDeviceCodeExpired) {
				w.WriteHeader(http.StatusGone)
				_ = json.NewEncoder(w).Encode(response{Success: false, Error: "expired_code"})
				return
			}

			if errors.Is(err, domain.ErrDeviceCodeUsed) {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(response{Success: false, Error: "code_already_used"})
				return
			}

			slog.Error("failed to authorize device", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response{Success: true})
	}
}

func (h *httpHandler) refreshToken() http.HandlerFunc {
	type request struct {
		RefreshToken string `json:"refresh_token"`
	}
	type response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		result, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
		if err != nil {
			if errors.Is(err, domain.ErrDeviceTokenNotFound) {
				http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
				return
			}
			if errors.Is(err, domain.ErrDeviceTokenRevoked) {
				http.Error(w, "Token has been revoked", http.StatusUnauthorized)
				return
			}
			slog.Error("failed to refresh token", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		resp := response{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresIn:    result.ExpiresIn,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (h *httpHandler) revokeToken() http.HandlerFunc {
	type response struct {
		Success bool `json:"success"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		accessToken := extractBearerToken(r)
		if accessToken == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		if err := h.svc.RevokeToken(r.Context(), accessToken); err != nil {
			slog.Error("failed to revoke token", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response{Success: true})
	}
}

func (h *httpHandler) getGCPCredentials() http.HandlerFunc {
	type response struct {
		ServiceAccountJSON string `json:"service_account_json"`
		ProjectID          string `json:"project_id,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, orgID, err := h.validateDeviceToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		integrations, err := h.integrationService.Integrations(ctx, backend.IntegrationsQuery{
			OrganizationID: orgID,
			ConnectorType:  backend.ConnectorTypeGCP,
			Status:         backend.IntegrationStatusActive,
		})
		if err != nil {
			slog.Error("failed to get integrations", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if len(integrations) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(httperrors.Error{
				Message:    "No GCP integration found",
				HttpStatus: http.StatusNotFound,
			})
			return
		}

		integration := integrations[0]

		// Get GCP credentials from integration credentials
		credentials, err := h.integrationService.IntegrationCredentials(ctx, backend.IntegrationCredentialsQuery{
			IntegrationID:  integration.ID,
			OrganizationID: orgID,
		})
		if err != nil {
			slog.Error("failed to fetch GCP credentials", "error", err)
			http.Error(w, "Failed to fetch credentials", http.StatusInternalServerError)
			return
		}

		serviceAccountJSON := credentials.Data["service_account_json"]
		projectID := integration.Metadata["project_id"]

		resp := response{
			ServiceAccountJSON: serviceAccountJSON,
			ProjectID:          projectID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (h *httpHandler) getGKEClusterInfo() http.HandlerFunc {
	type response struct {
		ClusterName string `json:"cluster_name"`
		Zone        string `json:"zone,omitempty"`
		Region      string `json:"region,omitempty"`
		ProjectID   string `json:"project_id"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, orgID, err := h.validateDeviceToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		integrations, err := h.integrationService.Integrations(ctx, backend.IntegrationsQuery{
			OrganizationID: orgID,
			ConnectorType:  backend.ConnectorTypeGCP,
			Status:         backend.IntegrationStatusActive,
		})
		if err != nil {
			slog.Error("failed to get integrations", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if len(integrations) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(httperrors.Error{
				Message:    "No GCP integration found",
				HttpStatus: http.StatusNotFound,
			})
			return
		}

		integration := integrations[0]

		// Extract GKE cluster info from integration metadata
		clusterName := integration.Metadata["gke_cluster_name"]
		zone := integration.Metadata["gke_cluster_zone"]
		region := integration.Metadata["gke_cluster_region"]
		projectID := integration.Metadata["project_id"]

		if clusterName == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(httperrors.Error{
				Message:    "No GKE cluster configured",
				HttpStatus: http.StatusNotFound,
			})
			return
		}

		resp := response{
			ClusterName: clusterName,
			Zone:        zone,
			Region:      region,
			ProjectID:   projectID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (h *httpHandler) validateDeviceToken(r *http.Request) (context.Context, uuid.UUID, error) {
	accessToken := extractBearerToken(r)
	if accessToken == "" {
		return nil, uuid.UUID{}, errors.New("missing authorization header")
	}

	result, err := h.svc.ValidateToken(r.Context(), accessToken)
	if err != nil {
		if errors.Is(err, domain.ErrDeviceTokenNotFound) {
			return nil, uuid.UUID{}, errors.New("invalid token")
		}
		if errors.Is(err, domain.ErrDeviceTokenRevoked) {
			return nil, uuid.UUID{}, errors.New("token has been revoked")
		}
		if errors.Is(err, domain.ErrDeviceTokenExpired) {
			return nil, uuid.UUID{}, errors.New("token has expired")
		}
		return nil, uuid.UUID{}, errors.New("token validation failed")
	}

	return r.Context(), result.OrganizationID, nil
}
