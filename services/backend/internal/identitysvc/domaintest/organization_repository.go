package domaintest

import (
	"context"
	"fmt"
	"sync"

	"github.com/73ai/infralayer/services/backend/internal/identitysvc/domain"
	"github.com/google/uuid"
)

type organizationRepository struct {
	mu       sync.RWMutex
	orgs     map[string]domain.Organization
	metadata map[uuid.UUID]domain.OrganizationMetadata
	members  map[string][]uuid.UUID
}

func NewOrganizationRepository() domain.OrganizationRepository {
	return &organizationRepository{
		orgs:     make(map[string]domain.Organization),
		metadata: make(map[uuid.UUID]domain.OrganizationMetadata),
		members:  make(map[string][]uuid.UUID),
	}
}

func (r *organizationRepository) Create(ctx context.Context, org domain.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orgs[org.ClerkOrgID]; exists {
		return fmt.Errorf("organization with clerk_org_id %s already exists", org.ClerkOrgID)
	}

	r.orgs[org.ClerkOrgID] = org
	return nil
}

func (r *organizationRepository) OrganizationByClerkID(ctx context.Context, clerkOrgID string) (*domain.Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	org, exists := r.orgs[clerkOrgID]
	if !exists {
		return nil, fmt.Errorf("organization with clerk_org_id %s not found", clerkOrgID)
	}

	if metadata, hasMetadata := r.metadata[org.ID]; hasMetadata {
		org.Metadata = metadata
	}

	return &org, nil
}

func (r *organizationRepository) OrganizationsByUserClerkID(ctx context.Context, clerkUserID string) ([]*domain.Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orgIDs, exists := r.members[clerkUserID]
	if !exists {
		return []*domain.Organization{}, nil
	}

	result := make([]*domain.Organization, 0, len(orgIDs))
	for _, orgID := range orgIDs {
		for _, org := range r.orgs {
			if org.ID == orgID {
				orgCopy := org
				if metadata, hasMetadata := r.metadata[org.ID]; hasMetadata {
					orgCopy.Metadata = metadata
				}
				result = append(result, &orgCopy)
				break
			}
		}
	}

	return result, nil
}

func (r *organizationRepository) Update(ctx context.Context, clerkOrgID string, org domain.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orgs[clerkOrgID]; !exists {
		return fmt.Errorf("organization with clerk_org_id %s not found", clerkOrgID)
	}

	existing := r.orgs[clerkOrgID]
	existing.Name = org.Name
	existing.Slug = org.Slug
	r.orgs[clerkOrgID] = existing

	return nil
}

func (r *organizationRepository) SetMetadata(ctx context.Context, organizationID uuid.UUID, metadata domain.OrganizationMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metadata[organizationID] = metadata
	return nil
}

func (r *organizationRepository) DeleteByClerkID(ctx context.Context, clerkOrgID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orgs[clerkOrgID]; !exists {
		return fmt.Errorf("organization with clerk_org_id %s not found", clerkOrgID)
	}

	delete(r.orgs, clerkOrgID)

	// Remove metadata and members associated with the organization
	for orgID, metadata := range r.metadata {
		if metadata.OrganizationID.String() == clerkOrgID {
			delete(r.metadata, orgID)
			break
		}
	}

	for userClerkID, orgIDs := range r.members {
		for i, orgID := range orgIDs {
			if orgID.String() == clerkOrgID {
				r.members[userClerkID] = append(orgIDs[:i], orgIDs[i+1:]...)
				break
			}
		}
	}

	return nil
}
