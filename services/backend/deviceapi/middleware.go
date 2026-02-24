package deviceapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/73ai/infralayer/services/backend/internal/devicesvc"
	"github.com/google/uuid"
)

type contextKey string

const (
	ContextKeyOrganizationID contextKey = "organization_id"
	ContextKeyUserID         contextKey = "user_id"
)

type DeviceTokenMiddleware struct {
	svc *devicesvc.Service
}

func NewDeviceTokenMiddleware(svc *devicesvc.Service) *DeviceTokenMiddleware {
	return &DeviceTokenMiddleware{svc: svc}
}

func (m *DeviceTokenMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken := extractBearerToken(r)
		if accessToken == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		result, err := m.svc.ValidateToken(r.Context(), accessToken)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ContextKeyOrganizationID, result.OrganizationID)
		ctx = context.WithValue(ctx, ContextKeyUserID, result.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}

func GetOrganizationID(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(ContextKeyOrganizationID)
	if val == nil {
		return uuid.UUID{}, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(ContextKeyUserID)
	if val == nil {
		return uuid.UUID{}, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}
