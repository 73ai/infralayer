package identitysvctest

import (
	"context"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/identitysvc"
	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"
	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domaintest"
)

func NewConfig() Config {
	return Config{
		Config: identitysvc.Config{},
	}
}

type Config struct {
	identitysvc.Config
}

type fixture struct {
	svc backend.IdentityService
}

func (f *fixture) Service() backend.IdentityService {
	return f.svc
}

func (c Config) Fixture() *fixture {
	return &fixture{
		svc: c.New(),
	}
}

func (c Config) New() backend.IdentityService {
	userRepo := domaintest.NewUserRepository()
	organizationRepo := domaintest.NewOrganizationRepository()
	memberRepo := domaintest.NewMemberRepository()

	return &service{
		userRepo:         userRepo,
		organizationRepo: organizationRepo,
		memberRepo:       memberRepo,
	}
}

type service struct {
	userRepo         domain.UserRepository
	organizationRepo domain.OrganizationRepository
	memberRepo       domain.MemberRepository
}

func (s *service) Subscribe(ctx context.Context) error {
	return nil
}

func (s *service) SubscribeUserCreated(ctx context.Context, event backend.UserCreatedEvent) error {
	return nil
}

func (s *service) SubscribeUserUpdated(ctx context.Context, event backend.UserUpdatedEvent) error {
	return nil
}

func (s *service) SubscribeUserDeleted(ctx context.Context, event backend.UserDeletedEvent) error {
	return nil
}

func (s *service) SubscribeOrganizationCreated(ctx context.Context, event backend.OrganizationCreatedEvent) error {
	return nil
}

func (s *service) SubscribeOrganizationUpdated(ctx context.Context, event backend.OrganizationUpdatedEvent) error {
	return nil
}

func (s *service) SubscribeOrganizationDeleted(ctx context.Context, event backend.OrganizationDeletedEvent) error {
	return nil
}

func (s *service) SubscribeOrganizationMemberAdded(ctx context.Context, event backend.OrganizationMemberAddedEvent) error {
	return nil
}

func (s *service) SubscribeOrganizationMemberUpdated(ctx context.Context, event backend.OrganizationMemberUpdatedEvent) error {
	return nil
}

func (s *service) SubscribeOrganizationMemberDeleted(ctx context.Context, event backend.OrganizationMemberDeletedEvent) error {
	return nil
}

func (s *service) SetOrganizationMetadata(ctx context.Context, cmd backend.OrganizationMetadataCommand) error {
	return nil
}

func (s *service) Profile(ctx context.Context, query backend.ProfileQuery) (backend.Profile, error) {
	// Mock implementation returns test data that matches the test expectations
	org, err := s.organizationRepo.OrganizationByClerkID(ctx, query.ClerkOrgID)
	if err != nil {
		return backend.Profile{}, err
	}

	user, err := s.userRepo.UserByClerkID(ctx, query.ClerkUserID)
	if err != nil {
		return backend.Profile{}, err
	}

	return backend.Profile{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt,
		Metadata: backend.OrganizationMetadata{
			OrganizationID:     org.Metadata.OrganizationID,
			CompanySize:        org.Metadata.CompanySize,
			TeamSize:           org.Metadata.TeamSize,
			UseCases:           org.Metadata.UseCases,
			ObservabilityStack: org.Metadata.ObservabilityStack,
			CompletedAt:        org.Metadata.CompletedAt,
			UpdatedAt:          org.Metadata.UpdatedAt,
		},
		OrganizationID: org.ID,
		UserID:         user.ID,
	}, nil
}
