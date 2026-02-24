package devicesvc

import (
	"database/sql"

	"github.com/73ai/infralayer/services/backend/internal/devicesvc/supporting/postgres"
)

type Config struct {
	Database *sql.DB `mapstructure:"-"`
}

func (c Config) New() *Service {
	deviceCodeRepo := postgres.NewDeviceCodeRepository(c.Database)
	deviceTokenRepo := postgres.NewDeviceTokenRepository(c.Database)

	return NewService(deviceCodeRepo, deviceTokenRepo)
}
