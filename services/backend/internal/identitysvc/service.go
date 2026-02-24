package identitysvc

import (
	"context"
	"fmt"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"
	"github.com/google/uuid"
)

type service struct {
	userRepo         domain.UserRepository
	organizationRepo domain.OrganizationRepository
	memberRepo       domain.MemberRepository
	authService      domain.AuthService
}

func (s *service) Subscribe(ctx context.Context) error {
	return s.authService.Subscribe(ctx, func(ctx context.Context, event any) error {
		switch e := event.(type) {
		case backend.UserCreatedEvent:
			return s.reconcileUserCreated(ctx, e)
		case backend.UserUpdatedEvent:
			return s.reconcileUserUpdated(ctx, e)
		case backend.UserDeletedEvent:
			return s.reconcileUserDeleted(ctx, e)
		case backend.OrganizationCreatedEvent:
			return s.reconcileOrganizationCreated(ctx, e)
		case backend.OrganizationUpdatedEvent:
			return s.reconcileOrganizationUpdated(ctx, e)
		case backend.OrganizationDeletedEvent:
			return s.reconcileOrganizationDeleted(ctx, e)
		case backend.OrganizationMemberAddedEvent:
			return s.reconcileOrganizationMemberAdded(ctx, e)
		case backend.OrganizationMemberUpdatedEvent:
			return s.reconcileOrganizationMemberUpdated(ctx, e)
		case backend.OrganizationMemberDeletedEvent:
			return s.reconcileOrganizationMemberDeleted(ctx, e)
		default:
			return fmt.Errorf("unknown event type: %T", e)
		}

	})
}

func (s *service) reconcileUserCreated(ctx context.Context, event backend.UserCreatedEvent) error {
	user := domain.User{
		ID:          uuid.New(),
		ClerkUserID: event.ClerkUserID,
		Email:       event.Email,
		FirstName:   event.FirstName,
		LastName:    event.LastName,
	}

	return s.userRepo.Create(ctx, user)
}

func (s *service) SubscribeUserCreated(ctx context.Context, event backend.UserCreatedEvent) error {
	panic("not allowed")
}

func (s *service) reconcileUserUpdated(ctx context.Context, event backend.UserUpdatedEvent) error {
	user := domain.User{
		Email:     event.Email,
		FirstName: event.FirstName,
		LastName:  event.LastName,
	}

	return s.userRepo.Update(ctx, event.ClerkUserID, user)
}

func (s *service) SubscribeUserUpdated(ctx context.Context, event backend.UserUpdatedEvent) error {
	panic("not allowed")
}

func (s *service) reconcileUserDeleted(ctx context.Context, event backend.UserDeletedEvent) error {
	return s.userRepo.DeleteByClerkID(ctx, event.ClerkUserID)
}

func (s *service) SubscribeUserDeleted(ctx context.Context, event backend.UserDeletedEvent) error {
	panic("not allowed")
}

func (s *service) reconcileOrganizationCreated(ctx context.Context, event backend.OrganizationCreatedEvent) error {
	createdByUser, err := s.userRepo.UserByClerkID(ctx, event.CreatedByUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	org := domain.Organization{
		ID:              uuid.New(),
		ClerkOrgID:      event.ClerkOrgID,
		Name:            event.Name,
		Slug:            event.Slug,
		CreatedByUserID: createdByUser.ID,
	}

	err = s.organizationRepo.Create(ctx, org)
	if err != nil {
		return fmt.Errorf("organization created: %w", err)
	}

	member := domain.OrganizationMember{
		UserID:         createdByUser.ID,
		OrganizationID: org.ID,
		ClerkUserID:    event.CreatedByUserID,
		ClerkOrgID:     event.ClerkOrgID,
		Role:           "admin",
	}

	return s.memberRepo.Create(ctx, member)
}

func (s *service) SubscribeOrganizationCreated(ctx context.Context, event backend.OrganizationCreatedEvent) error {
	panic("not allowed")
}

func (s *service) reconcileOrganizationUpdated(ctx context.Context, event backend.OrganizationUpdatedEvent) error {
	org := domain.Organization{
		Name: event.Name,
		Slug: event.Slug,
	}

	return s.organizationRepo.Update(ctx, event.ClerkOrgID, org)
}

func (s *service) SubscribeOrganizationUpdated(ctx context.Context, event backend.OrganizationUpdatedEvent) error {
	panic("not allowed")
}

func (s *service) reconcileOrganizationDeleted(ctx context.Context, event backend.OrganizationDeletedEvent) error {
	return s.organizationRepo.DeleteByClerkID(ctx, event.ClerkOrgID)
}

func (s *service) SubscribeOrganizationDeleted(ctx context.Context, event backend.OrganizationDeletedEvent) error {
	panic("not allowed")
}

func (s *service) reconcileOrganizationMemberAdded(ctx context.Context, event backend.OrganizationMemberAddedEvent) error {
	user, err := s.userRepo.UserByClerkID(ctx, event.ClerkUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	org, err := s.organizationRepo.OrganizationByClerkID(ctx, event.ClerkOrgID)
	if err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	member := domain.OrganizationMember{
		UserID:         user.ID,
		OrganizationID: org.ID,
		ClerkUserID:    event.ClerkUserID,
		ClerkOrgID:     event.ClerkOrgID,
		Role:           event.Role,
	}

	return s.memberRepo.Create(ctx, member)
}

func (s *service) SubscribeOrganizationMemberAdded(ctx context.Context, event backend.OrganizationMemberAddedEvent) error {
	panic("not allowed")
}

func (s *service) reconcileOrganizationMemberUpdated(ctx context.Context, event backend.OrganizationMemberUpdatedEvent) error {
	return s.memberRepo.UpdateByClerkIDs(ctx, event.ClerkUserID, event.ClerkOrgID, event.Role)
}

func (s *service) SubscribeOrganizationMemberUpdated(ctx context.Context, event backend.OrganizationMemberUpdatedEvent) error {
	panic("not allowed")
}

func (s *service) reconcileOrganizationMemberDeleted(ctx context.Context, event backend.OrganizationMemberDeletedEvent) error {
	return s.memberRepo.DeleteByClerkIDs(ctx, event.ClerkUserID, event.ClerkOrgID)
}

func (s *service) SubscribeOrganizationMemberDeleted(ctx context.Context, event backend.OrganizationMemberDeletedEvent) error {
	panic("not allowed")
}

func (s *service) SetOrganizationMetadata(ctx context.Context, cmd backend.OrganizationMetadataCommand) error {
	metadata := domain.OrganizationMetadata{
		OrganizationID:     cmd.OrganizationID,
		CompanySize:        cmd.CompanySize,
		TeamSize:           cmd.TeamSize,
		UseCases:           cmd.UseCases,
		ObservabilityStack: cmd.ObservabilityStack,
	}

	return s.organizationRepo.SetMetadata(ctx, cmd.OrganizationID, metadata)
}

func (s *service) Profile(ctx context.Context, query backend.ProfileQuery) (backend.Profile, error) {
	user, err := s.userRepo.UserByClerkID(ctx, query.ClerkUserID)
	if err != nil {
		return backend.Profile{}, fmt.Errorf("user not found: %w", err)
	}

	org, err := s.organizationRepo.OrganizationByClerkID(ctx, query.ClerkOrgID)
	if err != nil {
		return backend.Profile{}, fmt.Errorf("organization not found: %w", err)
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
