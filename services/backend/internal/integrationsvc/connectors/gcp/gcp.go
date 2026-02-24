package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/integrationsvc/domain"
	"github.com/google/uuid"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

type ServiceAccountKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain,omitempty"`
}

type ValidationResult struct {
	Valid       bool     `json:"valid"`
	ProjectID   string   `json:"project_id"`
	ClientEmail string   `json:"client_email"`
	HasViewer   bool     `json:"has_viewer_role"`
	Errors      []string `json:"errors,omitempty"`
}

type Connector struct {
	integrationRepository domain.IntegrationRepository
	credentialRepository  domain.CredentialRepository
}

func (c *Connector) InitiateAuthorization(organizationID string, userID string) (backend.IntegrationAuthorizationIntent, error) {
	return backend.IntegrationAuthorizationIntent{
		Type: backend.AuthorizationTypeAPIKey,
		URL:  "gcp-service-account",
	}, nil
}

func (c *Connector) ParseState(state string) (organizationID uuid.UUID, userID uuid.UUID, err error) {
	parts := strings.Split(state, ":")
	if len(parts) != 2 {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid state format")
	}

	orgID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid organization ID in state: %w", err)
	}

	uID, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid user ID in state: %w", err)
	}

	return orgID, uID, nil
}

func (c *Connector) CompleteAuthorization(authData backend.AuthorizationData) (backend.Credentials, error) {
	if authData.Code == "" {
		return backend.Credentials{}, fmt.Errorf("service account JSON is required")
	}

	var jsonCheck map[string]any
	if err := json.Unmarshal([]byte(authData.Code), &jsonCheck); err != nil {
		return backend.Credentials{}, fmt.Errorf("invalid JSON format")
	}

	creds := backend.Credentials{
		Type: backend.CredentialTypeServiceAccount,
		Data: map[string]string{
			"service_account_json": authData.Code,
		},
	}

	return creds, nil
}

func (c *Connector) ValidateCredentials(creds backend.Credentials) error {
	saJSON, exists := creds.Data["service_account_json"]
	if !exists {
		return fmt.Errorf("service account JSON not found in credentials")
	}

	validation, err := ValidateServiceAccountWithViewer([]byte(saJSON))
	if err != nil {
		return fmt.Errorf("credential validation failed - please check your service account JSON format and permissions")
	}

	if !validation.Valid {
		return fmt.Errorf("invalid service account - please check your credentials")
	}

	if !validation.HasViewer {
		return fmt.Errorf("service account does not have required Viewer permissions - please grant the Viewer role in the GCP Console")
	}

	return nil
}

func ValidateServiceAccountWithViewer(jsonData []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  false,
		Errors: []string{},
	}

	var sa ServiceAccountKey
	if err := json.Unmarshal(jsonData, &sa); err != nil {
		result.Errors = append(result.Errors, "invalid service account JSON format")
		return result, nil
	}

	if sa.Type != "service_account" {
		result.Errors = append(result.Errors, "invalid service account type")
		return result, nil
	}

	if sa.ProjectID == "" {
		result.Errors = append(result.Errors, "project_id is required")
		return result, nil
	}

	if sa.ClientEmail == "" {
		result.Errors = append(result.Errors, "client_email is required")
		return result, nil
	}

	if sa.PrivateKey == "" {
		result.Errors = append(result.Errors, "private_key is required")
		return result, nil
	}

	result.ProjectID = sa.ProjectID
	result.ClientEmail = sa.ClientEmail

	ctx := context.Background()
	option := option.WithCredentialsJSON(jsonData)
	service, err := cloudresourcemanager.NewService(ctx, option)
	if err != nil {
		result.Errors = append(result.Errors, "failed to authenticate with service account")
		return result, nil
	}

	project, err := service.Projects.Get(sa.ProjectID).Context(ctx).Do()
	if err != nil {
		result.Errors = append(result.Errors, "failed to access project - please verify permissions")
		return result, nil
	}

	if project == nil {
		result.Errors = append(result.Errors, "project not found or no access")
		return result, nil
	}

	policy, err := service.Projects.GetIamPolicy(sa.ProjectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		result.Errors = append(result.Errors, "failed to retrieve IAM policy")
		return result, nil
	}

	hasViewer := false
	viewerRoles := []string{"roles/viewer", "roles/owner", "roles/editor"}
	memberIdentity := fmt.Sprintf("serviceAccount:%s", sa.ClientEmail)

	for _, binding := range policy.Bindings {
		for _, role := range viewerRoles {
			if binding.Role == role {
				for _, member := range binding.Members {
					if member == memberIdentity {
						hasViewer = true
						break
					}
				}
			}
			if hasViewer {
				break
			}
		}
		if hasViewer {
			break
		}
	}

	if !hasViewer {
		result.Errors = append(result.Errors, "service account does not have Viewer role")
	}

	result.HasViewer = hasViewer
	result.Valid = len(result.Errors) == 0

	return result, nil
}

func (c *Connector) RefreshCredentials(creds backend.Credentials) (backend.Credentials, error) {
	return creds, nil
}

func (c *Connector) RevokeCredentials(creds backend.Credentials) error {
	return nil
}

func (c *Connector) ConfigureWebhooks(integrationID string, creds backend.Credentials) error {
	return nil
}

func (c *Connector) ValidateWebhookSignature(payload []byte, signature string, secret string) error {
	return fmt.Errorf("webhooks not supported for GCP connector")
}

func (c *Connector) Subscribe(ctx context.Context, handler func(ctx context.Context, event any) error) error {
	<-ctx.Done()
	return ctx.Err()
}

func (c *Connector) ProcessEvent(ctx context.Context, event any) error {
	return fmt.Errorf("event processing not supported for GCP connector")
}

func (c *Connector) Sync(ctx context.Context, integration backend.Integration, params map[string]string) error {
	credRecord, err := c.credentialRepository.FindByIntegration(ctx, integration.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	creds := backend.Credentials{
		Type:      credRecord.CredentialType,
		Data:      credRecord.Data,
		ExpiresAt: credRecord.ExpiresAt,
	}

	return c.ValidateCredentials(creds)
}
