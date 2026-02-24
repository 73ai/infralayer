package postgres

import (
	"database/sql"

	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
)

type Config struct {
	DB *sql.DB
}

func (c Config) NewIntegrationRepository() domain.IntegrationRepository {
	return NewIntegrationRepository(c.DB)
}

func (c Config) NewCredentialRepository() (domain.CredentialRepository, error) {
	return NewCredentialRepository(c.DB)
}

type IntegrationDB struct {
	db *sql.DB
	*Queries
}

func NewIntegrationDB(db *sql.DB) *IntegrationDB {
	return &IntegrationDB{
		db:      db,
		Queries: New(db),
	}
}

func (i *IntegrationDB) DB() *sql.DB {
	return i.db
}

func (i *IntegrationDB) Close() error {
	return i.db.Close()
}
