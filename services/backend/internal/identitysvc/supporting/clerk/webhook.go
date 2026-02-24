package clerk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/73ai/infralayer/services/backend/internal/generic/httperrors"
	svix "github.com/svix/svix-webhooks/go"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"
)

type user struct {
	ID             string `json:"id"`
	EmailAddresses []struct {
		EmailAddress string `json:"email_address"`
	} `json:"email_addresses"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type organization struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedBy string `json:"created_by"`
}

type organizationMembership struct {
	Organization struct {
		ID string `json:"id"`
	} `json:"organization"`
	PublicUserData struct {
		UserID string `json:"user_id"`
	} `json:"public_user_data"`
	Role string `json:"role"`
}

type webhookHandler struct {
	http.ServeMux
	callbackHandlerFunc func(ctx context.Context, event any) error
}

type webhookServerConfig struct {
	port                int
	webhookSecret       string
	callbackHandlerFunc func(ctx context.Context, event any) error
}

func (c webhookServerConfig) startWebhookServer(ctx context.Context) error {
	wh, err := svix.NewWebhook(c.webhookSecret)
	if err != nil {
		panic(fmt.Errorf("error creating svix webhook: %w", err))
	}

	h := &webhookHandler{
		callbackHandlerFunc: c.callbackHandlerFunc,
	}
	h.init()

	httpServer := &http.Server{
		Addr:        fmt.Sprintf(":%d", c.port),
		BaseContext: func(net.Listener) context.Context { return ctx },
		Handler:     panicMiddleware(webhookValidationMiddleware(wh, h)),
	}

	return httpServer.ListenAndServe()
}

func (wh *webhookHandler) init() {
	wh.HandleFunc("/webhooks/clerk", wh.handler())
}

func (wh *webhookHandler) handler() func(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	type response struct{}

	return ApiHandlerFunc(func(ctx context.Context, r request) (response, error) {
		switch r.Type {
		case "user.created":
			return response{}, wh.handleUserCreated(ctx, r.Data)
		case "user.updated":
			return response{}, wh.handleUserUpdated(ctx, r.Data)
		case "user.deleted":
			return response{}, wh.handleUserDeleted(ctx, r.Data)
		case "organization.created":
			return response{}, wh.handleOrganizationCreated(ctx, r.Data)
		case "organization.updated":
			return response{}, wh.handleOrganizationUpdated(ctx, r.Data)
		case "organization.deleted":
			return response{}, wh.handleOrganizationDeleted(ctx, r.Data)
		case "organizationMembership.created":
			return response{}, wh.handleOrganizationMemberAdded(ctx, r.Data)
		case "organizationMembership.deleted":
			return response{}, wh.handleOrganizationMemberRemoved(ctx, r.Data)
		case "organizationMembership.updated":
			return response{}, wh.handleOrganizationMemberUpdated(ctx, r.Data)
		default:
			return response{}, fmt.Errorf("unsupported webhook event type: %s", r.Type)
		}
	})
}

func (wh *webhookHandler) handleUserCreated(ctx context.Context, data json.RawMessage) error {
	var user user
	if err := json.Unmarshal(data, &user); err != nil {
		return fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	email := ""
	if len(user.EmailAddresses) > 0 {
		email = user.EmailAddresses[0].EmailAddress
	}

	event := backend.UserCreatedEvent{
		ClerkUserID: user.ID,
		Email:       email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
	}

	return wh.callbackHandlerFunc(ctx, event)
}

func (wh *webhookHandler) handleUserUpdated(ctx context.Context, data json.RawMessage) error {
	var user user
	if err := json.Unmarshal(data, &user); err != nil {
		return fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	email := ""
	if len(user.EmailAddresses) > 0 {
		email = user.EmailAddresses[0].EmailAddress
	}

	event := backend.UserUpdatedEvent{
		ClerkUserID: user.ID,
		Email:       email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
	}

	return wh.callbackHandlerFunc(ctx, event)
}

func (wh *webhookHandler) handleUserDeleted(ctx context.Context, data json.RawMessage) error {
	var user user
	if err := json.Unmarshal(data, &user); err != nil {
		return fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	event := backend.UserDeletedEvent{
		ClerkUserID: user.ID,
	}

	return wh.callbackHandlerFunc(ctx, event)
}

func (wh *webhookHandler) handleOrganizationCreated(ctx context.Context, data json.RawMessage) error {
	var org organization
	if err := json.Unmarshal(data, &org); err != nil {
		return fmt.Errorf("failed to unmarshal organization data: %w", err)
	}

	event := backend.OrganizationCreatedEvent{
		ClerkOrgID:      org.ID,
		Name:            org.Name,
		Slug:            org.Slug,
		CreatedByUserID: org.CreatedBy,
	}

	return wh.callbackHandlerFunc(ctx, event)
}

func (wh *webhookHandler) handleOrganizationUpdated(ctx context.Context, data json.RawMessage) error {
	var org organization
	if err := json.Unmarshal(data, &org); err != nil {
		return fmt.Errorf("failed to unmarshal organization data: %w", err)
	}

	event := backend.OrganizationUpdatedEvent{
		ClerkOrgID: org.ID,
		Name:       org.Name,
		Slug:       org.Slug,
	}

	return wh.callbackHandlerFunc(ctx, event)
}

func (wh *webhookHandler) handleOrganizationDeleted(ctx context.Context, data json.RawMessage) error {
	var org organization
	if err := json.Unmarshal(data, &org); err != nil {
		return fmt.Errorf("failed to unmarshal organization data: %w", err)
	}

	event := backend.OrganizationDeletedEvent{
		ClerkOrgID: org.ID,
	}

	return wh.callbackHandlerFunc(ctx, event)
}

func (wh *webhookHandler) handleOrganizationMemberAdded(ctx context.Context, data json.RawMessage) error {
	var membership organizationMembership
	if err := json.Unmarshal(data, &membership); err != nil {
		return fmt.Errorf("failed to unmarshal membership data: %w", err)
	}

	event := backend.OrganizationMemberAddedEvent{
		ClerkUserID: membership.PublicUserData.UserID,
		ClerkOrgID:  membership.Organization.ID,
		Role:        membership.Role,
	}

	return wh.callbackHandlerFunc(ctx, event)
}

func (wh *webhookHandler) handleOrganizationMemberRemoved(ctx context.Context, data json.RawMessage) error {
	var membership organizationMembership
	if err := json.Unmarshal(data, &membership); err != nil {
		return fmt.Errorf("failed to unmarshal membership data: %w", err)
	}

	event := backend.OrganizationMemberDeletedEvent{
		ClerkUserID: membership.PublicUserData.UserID,
		ClerkOrgID:  membership.Organization.ID,
	}

	return wh.callbackHandlerFunc(ctx, event)
}

func (wh *webhookHandler) handleOrganizationMemberUpdated(ctx context.Context, data json.RawMessage) error {
	var membership organizationMembership
	if err := json.Unmarshal(data, &membership); err != nil {
		return fmt.Errorf("failed to unmarshal membership data: %w", err)
	}

	event := backend.OrganizationMemberUpdatedEvent{
		ClerkUserID: membership.PublicUserData.UserID,
		ClerkOrgID:  membership.Organization.ID,
		Role:        membership.Role,
	}

	return wh.callbackHandlerFunc(ctx, event)
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
			// Return 200 OK for duplicate key errors to prevent Clerk from retrying
			if errors.Is(err, domain.ErrDuplicateKey) {
				slog.Info("clerk webhook: duplicate key error, returning 200 OK", "path", r.URL, "err", err)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(response)
				return
			}

			slog.Error("error in clerk webhook api handler", "path", r.URL, "err", err)
			var httpError = httperrors.From(err)
			w.WriteHeader(httpError.HttpStatus)
			_ = json.NewEncoder(w).Encode(httpError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

func panicMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("clerk: panic while handling http request", "recover", r)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func webhookValidationMiddleware(webhook *svix.Webhook, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		err = webhook.Verify(body, r.Header)
		if err != nil {
			slog.Info("clerk: webhook validation failed", "error", err, "headers", r.Header)
			http.Error(w, "Invalid webhook signature", http.StatusUnauthorized)
			return
		}

		// Restore body for downstream handlers
		r.Body = io.NopCloser(strings.NewReader(string(body)))

		next.ServeHTTP(w, r)
	})
}
