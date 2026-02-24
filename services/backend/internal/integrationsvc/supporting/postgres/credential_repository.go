package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
	"github.com/google/uuid"
)

type credentialRepository struct {
	queries    *Queries
	encryption *encryptionService
}

func NewCredentialRepository(sqlDB *sql.DB) (domain.CredentialRepository, error) {
	encryption, err := newEncryptionService()
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption service: %w", err)
	}

	return &credentialRepository{
		queries:    New(sqlDB),
		encryption: encryption,
	}, nil
}

func (r *credentialRepository) Store(ctx context.Context, cred domain.IntegrationCredential) error {
	encryptedData, err := r.encryption.encrypt(cred.Data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential data: %w", err)
	}

	credentialID := cred.ID
	integrationID := cred.IntegrationID

	var expiresAt sql.NullTime
	if cred.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *cred.ExpiresAt, Valid: true}
	}

	return r.queries.StoreCredential(ctx, StoreCredentialParams{
		ID:                      credentialID,
		IntegrationID:           integrationID,
		CredentialType:          string(cred.CredentialType),
		CredentialDataEncrypted: encryptedData,
		ExpiresAt:               expiresAt,
		EncryptionKeyID:         cred.EncryptionKeyID,
		CreatedAt:               cred.CreatedAt,
		UpdatedAt:               cred.UpdatedAt,
	})
}

func (r *credentialRepository) FindByIntegration(ctx context.Context, integrationID uuid.UUID) (domain.IntegrationCredential, error) {
	dbCredential, err := r.queries.FindCredentialByIntegration(ctx, integrationID)
	if err != nil {
		return domain.IntegrationCredential{}, fmt.Errorf("failed to find credential: %w", err)
	}

	return r.mapToCredential(dbCredential)
}

func (r *credentialRepository) Update(ctx context.Context, cred domain.IntegrationCredential) error {
	encryptedData, err := r.encryption.encrypt(cred.Data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential data: %w", err)
	}

	integrationID := cred.IntegrationID

	var expiresAt sql.NullTime
	if cred.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *cred.ExpiresAt, Valid: true}
	}

	return r.queries.UpdateCredential(ctx, UpdateCredentialParams{
		IntegrationID:           integrationID,
		CredentialType:          string(cred.CredentialType),
		CredentialDataEncrypted: encryptedData,
		ExpiresAt:               expiresAt,
		EncryptionKeyID:         cred.EncryptionKeyID,
	})
}

func (r *credentialRepository) Delete(ctx context.Context, integrationID uuid.UUID) error {

	return r.queries.DeleteCredential(ctx, integrationID)
}

func (r *credentialRepository) FindExpiring(ctx context.Context, before time.Time) ([]domain.IntegrationCredential, error) {
	dbCredentials, err := r.queries.FindExpiringCredentials(ctx, sql.NullTime{Time: before, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to find expiring credentials: %w", err)
	}

	credentials := make([]domain.IntegrationCredential, len(dbCredentials))
	for i, dbCredential := range dbCredentials {
		credential, err := r.mapToCredential(dbCredential)
		if err != nil {
			return nil, fmt.Errorf("failed to map credential: %w", err)
		}
		credentials[i] = credential
	}

	return credentials, nil
}

func (r *credentialRepository) mapToCredential(dbCredential IntegrationCredential) (domain.IntegrationCredential, error) {
	decryptedData, err := r.encryption.decrypt(dbCredential.CredentialDataEncrypted)
	if err != nil {
		return domain.IntegrationCredential{}, fmt.Errorf("failed to decrypt credential data: %w", err)
	}

	var expiresAt *time.Time
	if dbCredential.ExpiresAt.Valid {
		expiresAt = &dbCredential.ExpiresAt.Time
	}

	return domain.IntegrationCredential{
		ID:              dbCredential.ID,
		IntegrationID:   dbCredential.IntegrationID,
		CredentialType:  backend.CredentialType(dbCredential.CredentialType),
		Data:            decryptedData,
		ExpiresAt:       expiresAt,
		EncryptionKeyID: dbCredential.EncryptionKeyID,
		CreatedAt:       dbCredential.CreatedAt,
		UpdatedAt:       dbCredential.UpdatedAt,
	}, nil
}
