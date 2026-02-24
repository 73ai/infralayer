package identitytest

import (
	"context"
	"testing"

	"github.com/73ai/infralayer/services/backend"
)

func Ensure(t *testing.T, f fixture) {
	t.Run("SubscribeUserCreated", func(t *testing.T) {
		t.Run("creates user successfully", func(t *testing.T) {
			ctx := context.Background()

			event := backend.UserCreatedEvent{
				ClerkUserID: "user_test123",
				Email:       "test@example.com",
				FirstName:   "John",
				LastName:    "Doe",
			}

			err := f.Service().SubscribeUserCreated(ctx, event)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	})

	t.Run("Organization", func(t *testing.T) {
		t.Run("full workflow with metadata", func(t *testing.T) {
			ctx := context.Background()
			svc := f.Service()

			// 1. Create user
			userEvent := backend.UserCreatedEvent{
				ClerkUserID: "user_workflow123",
				Email:       "workflow@example.com",
				FirstName:   "Jane",
				LastName:    "Smith",
			}
			err := svc.SubscribeUserCreated(ctx, userEvent)
			if err != nil {
				t.Fatalf("failed to create user: %v", err)
			}

			// 2. Create organization
			orgEvent := backend.OrganizationCreatedEvent{
				ClerkOrgID:      "org_workflow123",
				Name:            "Test Org",
				Slug:            "test-org",
				CreatedByUserID: "user_workflow123",
			}
			err = svc.SubscribeOrganizationCreated(ctx, orgEvent)
			if err != nil {
				t.Fatalf("failed to create organization: %v", err)
			}

			// 3. Get profile without metadata
			query := backend.ProfileQuery{
				ClerkOrgID:  "org_workflow123",
				ClerkUserID: "user_workflow123",
			}
			profile, err := svc.Profile(ctx, query)
			if err != nil {
				t.Fatalf("failed to get profile: %v", err)
			}

			if profile.Name != "Test Org" {
				t.Errorf("expected org name 'Test Org', got '%s'", profile.Name)
			}

			// 4. Set metadata
			cmd := backend.OrganizationMetadataCommand{
				OrganizationID:     profile.ID,
				CompanySize:        backend.CompanySizeStartup,
				TeamSize:           backend.TeamSize1To5,
				UseCases:           []backend.UseCase{backend.UseCaseInfrastructureMonitoring},
				ObservabilityStack: []backend.ObservabilityStack{backend.ObservabilityStackDatadog},
			}
			err = svc.SetOrganizationMetadata(ctx, cmd)
			if err != nil {
				t.Fatalf("failed to set metadata: %v", err)
			}

			// 5. Get profile with metadata
			profile, err = svc.Profile(ctx, query)
			if err != nil {
				t.Fatalf("failed to get profile with metadata: %v", err)
			}

			if profile.Metadata.CompanySize != backend.CompanySizeStartup {
				t.Errorf("expected company size '%s', got '%s'", backend.CompanySizeStartup, profile.Metadata.CompanySize)
			}

			if len(profile.Metadata.UseCases) != 1 || profile.Metadata.UseCases[0] != backend.UseCaseInfrastructureMonitoring {
				t.Errorf("expected use cases [%s], got %v", backend.UseCaseInfrastructureMonitoring, profile.Metadata.UseCases)
			}
		})
	})

	t.Run("setOrganizationMetadata", func(t *testing.T) {
		t.Run("sets metadata successfully", func(t *testing.T) {
			t.Skip("skipping - needs organization setup")
		})
	})
}

type fixture interface {
	Service() backend.IdentityService
}
