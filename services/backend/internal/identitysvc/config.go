package identitysvc

import (
	"database/sql"

	"github.com/73ai/infralayer/services/backend/internal/identitysvc/supporting/clerk"

	"github.com/73ai/infralayer/services/backend/internal/identitysvc/supporting/postgres"
)

type Config struct {
	Database *sql.DB      `mapstructure:"-"`
	Clerk    clerk.Config `mapstructure:"clerk"`
}

func (c Config) New(db *sql.DB) *service {
	userRepo := postgres.NewUserRepository(db)
	organizationRepo := postgres.NewOrganizationRepository(db)
	memberRepo := postgres.NewMemberRepository(db)

	return &service{
		userRepo:         userRepo,
		organizationRepo: organizationRepo,
		memberRepo:       memberRepo,
		authService:      c.Clerk.NewAuthService(),
	}
}
