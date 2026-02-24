package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/73ai/infralayer/services/backend/internal/devicesvc/domain"
	"github.com/google/uuid"
)

type deviceCodeRepository struct {
	queries *Queries
}

func NewDeviceCodeRepository(sqlDB *sql.DB) domain.DeviceCodeRepository {
	return &deviceCodeRepository{
		queries: New(sqlDB),
	}
}

func (r *deviceCodeRepository) Create(ctx context.Context, code domain.DeviceCode) error {
	return r.queries.CreateDeviceCode(ctx, CreateDeviceCodeParams{
		ID:         code.ID,
		DeviceCode: code.DeviceCode,
		UserCode:   code.UserCode,
		Status:     string(code.Status),
		ExpiresAt:  code.ExpiresAt,
		CreatedAt:  code.CreatedAt,
	})
}

func (r *deviceCodeRepository) GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceCode, error) {
	dbCode, err := r.queries.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeviceCodeNotFound
		}
		return nil, err
	}

	return r.mapToDomain(dbCode), nil
}

func (r *deviceCodeRepository) GetByDeviceCode(ctx context.Context, deviceCode string) (*domain.DeviceCode, error) {
	dbCode, err := r.queries.GetDeviceCodeByDeviceCode(ctx, deviceCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeviceCodeNotFound
		}
		return nil, err
	}

	return r.mapToDomain(dbCode), nil
}

func (r *deviceCodeRepository) Authorize(ctx context.Context, userCode string, organizationID, userID uuid.UUID) error {
	return r.queries.AuthorizeDeviceCode(ctx, AuthorizeDeviceCodeParams{
		UserCode:       userCode,
		OrganizationID: uuid.NullUUID{UUID: organizationID, Valid: true},
		UserID:         uuid.NullUUID{UUID: userID, Valid: true},
	})
}

func (r *deviceCodeRepository) MarkAsUsed(ctx context.Context, deviceCode string) error {
	return r.queries.MarkDeviceCodeAsUsed(ctx, deviceCode)
}

func (r *deviceCodeRepository) DeleteExpired(ctx context.Context) error {
	return r.queries.DeleteExpiredDeviceCodes(ctx)
}

func (r *deviceCodeRepository) mapToDomain(dbCode DeviceCode) *domain.DeviceCode {
	code := &domain.DeviceCode{
		ID:         dbCode.ID,
		DeviceCode: dbCode.DeviceCode,
		UserCode:   dbCode.UserCode,
		Status:     domain.DeviceCodeStatus(dbCode.Status),
		ExpiresAt:  dbCode.ExpiresAt,
		CreatedAt:  dbCode.CreatedAt,
	}

	if dbCode.OrganizationID.Valid {
		code.OrganizationID = dbCode.OrganizationID.UUID
	}
	if dbCode.UserID.Valid {
		code.UserID = dbCode.UserID.UUID
	}

	if code.ExpiresAt.Before(time.Now()) && code.Status == domain.DeviceCodeStatusPending {
		code.Status = domain.DeviceCodeStatusExpired
	}

	return code
}
