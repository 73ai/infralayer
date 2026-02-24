package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/73ai/infralayer/services/backend/internal/devicesvc/domain"
	"github.com/google/uuid"
)

type deviceTokenRepository struct {
	queries *Queries
}

func NewDeviceTokenRepository(sqlDB *sql.DB) domain.DeviceTokenRepository {
	return &deviceTokenRepository{
		queries: New(sqlDB),
	}
}

func (r *deviceTokenRepository) Create(ctx context.Context, token domain.DeviceToken) error {
	return r.queries.CreateDeviceToken(ctx, CreateDeviceTokenParams{
		ID:             token.ID,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		OrganizationID: token.OrganizationID,
		UserID:         token.UserID,
		DeviceName:     sql.NullString{String: token.DeviceName, Valid: token.DeviceName != ""},
		ExpiresAt:      token.ExpiresAt,
		CreatedAt:      token.CreatedAt,
	})
}

func (r *deviceTokenRepository) GetByAccessToken(ctx context.Context, accessToken string) (*domain.DeviceToken, error) {
	dbToken, err := r.queries.GetDeviceTokenByAccessToken(ctx, accessToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeviceTokenNotFound
		}
		return nil, err
	}

	return r.mapToDomain(dbToken), nil
}

func (r *deviceTokenRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.DeviceToken, error) {
	dbToken, err := r.queries.GetDeviceTokenByRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeviceTokenNotFound
		}
		return nil, err
	}

	return r.mapToDomain(dbToken), nil
}

func (r *deviceTokenRepository) Revoke(ctx context.Context, accessToken string) error {
	return r.queries.RevokeDeviceToken(ctx, accessToken)
}

func (r *deviceTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.queries.RevokeAllDeviceTokensForUser(ctx, userID)
}

func (r *deviceTokenRepository) UpdateTokens(ctx context.Context, oldRefreshToken string, token domain.DeviceToken) error {
	return r.queries.UpdateDeviceTokens(ctx, UpdateDeviceTokensParams{
		RefreshToken:   oldRefreshToken,
		AccessToken:    token.AccessToken,
		RefreshToken_2: token.RefreshToken,
		ExpiresAt:      token.ExpiresAt,
	})
}

func (r *deviceTokenRepository) mapToDomain(dbToken DeviceToken) *domain.DeviceToken {
	token := &domain.DeviceToken{
		ID:             dbToken.ID,
		AccessToken:    dbToken.AccessToken,
		RefreshToken:   dbToken.RefreshToken,
		OrganizationID: dbToken.OrganizationID,
		UserID:         dbToken.UserID,
		ExpiresAt:      dbToken.ExpiresAt,
		CreatedAt:      dbToken.CreatedAt,
	}

	if dbToken.DeviceName.Valid {
		token.DeviceName = dbToken.DeviceName.String
	}

	if dbToken.RevokedAt.Valid {
		token.RevokedAt = &dbToken.RevokedAt.Time
	}

	return token
}
