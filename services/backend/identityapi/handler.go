package identityapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/73ai/infralayer/services/backend/internal/generic/httperrors"

	"github.com/73ai/infralayer/services/backend"
	"github.com/google/uuid"
)

type httpHandler struct {
	http.ServeMux
	svc backend.IdentityService
}

func (h *httpHandler) init() {
	h.HandleFunc("/identity/organization/", h.organization())
	h.HandleFunc("/identity/me/", h.me())
	h.HandleFunc("/identity/organization/set-metadata/", h.setOrganizationMetadata())
}

func NewHandler(identityService backend.IdentityService,
	authMiddleware func(handler http.Handler) http.Handler) http.Handler {
	h := &httpHandler{
		svc: identityService,
	}

	h.init()
	return authMiddleware(h)
}

func (h *httpHandler) organization() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		ClerkOrgID  string `json:"clerk_org_id"`
		ClerkUserID string `json:"clerk_user_id"`
	}
	type response struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Slug           string `json:"slug"`
		CreatedAt      string `json:"created_at"`
		OrganizationID string `json:"organization_id"`
		UserID         string `json:"user_id"`
		Metadata       struct {
			CompanySize        string   `json:"company_size"`
			TeamSize           string   `json:"team_size"`
			UseCases           []string `json:"use_cases"`
			ObservabilityStack []string `json:"observability_stack"`
			CompletedAt        string   `json:"completed_at"`
		} `json:"metadata"`
	}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		query := backend.ProfileQuery{
			ClerkOrgID:  req.ClerkOrgID,
			ClerkUserID: req.ClerkUserID,
		}

		profile, err := h.svc.Profile(ctx, query)
		if err != nil {
			return response{}, err
		}

		useCases := make([]string, len(profile.Metadata.UseCases))
		for i, uc := range profile.Metadata.UseCases {
			useCases[i] = string(uc)
		}

		stack := make([]string, len(profile.Metadata.ObservabilityStack))
		for i, s := range profile.Metadata.ObservabilityStack {
			stack[i] = string(s)
		}

		resp := response{
			ID:             profile.ID.String(),
			Name:           profile.Name,
			Slug:           profile.Slug,
			CreatedAt:      profile.CreatedAt.Format(time.RFC3339),
			OrganizationID: profile.OrganizationID.String(),
			UserID:         profile.UserID.String(),
			Metadata: struct {
				CompanySize        string   `json:"company_size"`
				TeamSize           string   `json:"team_size"`
				UseCases           []string `json:"use_cases"`
				ObservabilityStack []string `json:"observability_stack"`
				CompletedAt        string   `json:"completed_at"`
			}{
				CompanySize:        string(profile.Metadata.CompanySize),
				TeamSize:           string(profile.Metadata.TeamSize),
				UseCases:           useCases,
				ObservabilityStack: stack,
				CompletedAt:        profile.Metadata.CompletedAt.Format(time.RFC3339),
			},
		}

		return resp, nil
	})
}

func (h *httpHandler) me() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		ClerkOrgID  string `json:"clerk_org_id"`
		ClerkUserID string `json:"clerk_user_id"`
	}
	type response struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Slug           string `json:"slug"`
		CreatedAt      string `json:"created_at"`
		OrganizationID string `json:"organization_id"`
		UserID         string `json:"user_id"`
		Metadata       struct {
			CompanySize        string   `json:"company_size"`
			TeamSize           string   `json:"team_size"`
			UseCases           []string `json:"use_cases"`
			ObservabilityStack []string `json:"observability_stack"`
			CompletedAt        string   `json:"completed_at"`
		} `json:"metadata"`
	}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		query := backend.ProfileQuery{
			ClerkOrgID:  req.ClerkOrgID,
			ClerkUserID: req.ClerkUserID,
		}

		profile, err := h.svc.Profile(ctx, query)
		if err != nil {
			return response{}, err
		}

		useCases := make([]string, len(profile.Metadata.UseCases))
		for i, uc := range profile.Metadata.UseCases {
			useCases[i] = string(uc)
		}

		stack := make([]string, len(profile.Metadata.ObservabilityStack))
		for i, s := range profile.Metadata.ObservabilityStack {
			stack[i] = string(s)
		}

		resp := response{
			ID:             profile.ID.String(),
			Name:           profile.Name,
			Slug:           profile.Slug,
			CreatedAt:      profile.CreatedAt.Format(time.RFC3339),
			OrganizationID: profile.OrganizationID.String(),
			UserID:         profile.UserID.String(),
			Metadata: struct {
				CompanySize        string   `json:"company_size"`
				TeamSize           string   `json:"team_size"`
				UseCases           []string `json:"use_cases"`
				ObservabilityStack []string `json:"observability_stack"`
				CompletedAt        string   `json:"completed_at"`
			}{
				CompanySize:        string(profile.Metadata.CompanySize),
				TeamSize:           string(profile.Metadata.TeamSize),
				UseCases:           useCases,
				ObservabilityStack: stack,
				CompletedAt:        profile.Metadata.CompletedAt.Format(time.RFC3339),
			},
		}

		return resp, nil
	})
}

func (h *httpHandler) setOrganizationMetadata() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		OrganizationID     string   `json:"organization_id"`
		CompanySize        string   `json:"company_size"`
		TeamSize           string   `json:"team_size"`
		UseCases           []string `json:"use_cases"`
		ObservabilityStack []string `json:"observability_stack"`
	}
	type response struct{}

	return ApiHandlerFunc(func(ctx context.Context, req request) (response, error) {
		orgID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			return response{}, err
		}

		useCases := make([]backend.UseCase, len(req.UseCases))
		for i, uc := range req.UseCases {
			useCases[i] = backend.UseCase(uc)
		}

		stack := make([]backend.ObservabilityStack, len(req.ObservabilityStack))
		for i, s := range req.ObservabilityStack {
			stack[i] = backend.ObservabilityStack(s)
		}

		cmd := backend.OrganizationMetadataCommand{
			OrganizationID:     orgID,
			CompanySize:        backend.CompanySize(req.CompanySize),
			TeamSize:           backend.TeamSize(req.TeamSize),
			UseCases:           useCases,
			ObservabilityStack: stack,
		}

		err = h.svc.SetOrganizationMetadata(ctx, cmd)
		return response{}, err
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
			slog.Error("error in identity api handler", "path", r.URL, "request", request, "err", err)
			var httpError = httperrors.From(err)
			w.WriteHeader(httpError.HttpStatus)
			_ = json.NewEncoder(w).Encode(httpError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}
