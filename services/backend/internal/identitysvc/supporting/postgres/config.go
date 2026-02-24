package postgres

import (
	"github.com/73ai/infralayer/services/backend/internal/generic/postgresconfig"
	_ "github.com/lib/pq"
)

type Config struct {
	postgresconfig.Config
}

func (c Config) New() (*IdentityDB, error) {
	db, err := c.Init()
	if err != nil {
		return nil, err
	}

	return &IdentityDB{
		db:      db,
		Querier: New(db),
	}, nil
}
