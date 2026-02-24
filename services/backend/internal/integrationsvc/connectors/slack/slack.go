package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/73ai/infralayer/services/backend"
	"github.com/google/uuid"
	"github.com/slack-go/slack"
)

type slackConnector struct {
	config Config
	client *http.Client
}

func (s *slackConnector) InitiateAuthorization(organizationID string, userID string) (backend.IntegrationAuthorizationIntent, error) {
	state := fmt.Sprintf("%s:%s:%d", organizationID, userID, time.Now().Unix())

	params := url.Values{}
	params.Set("client_id", s.config.ClientID)
	params.Set("scope", strings.Join(s.config.Scopes, ","))
	params.Set("redirect_uri", s.config.RedirectURL)
	params.Set("state", state)
	params.Set("user_scope", "")

	authURL := fmt.Sprintf("https://slack.com/oauth/v2/authorize?%s", params.Encode())

	return backend.IntegrationAuthorizationIntent{
		Type: backend.AuthorizationTypeOAuth2,
		URL:  authURL,
	}, nil
}

func (s *slackConnector) ParseState(state string) (organizationID uuid.UUID, userID uuid.UUID, err error) {
	parts := strings.Split(state, ":")
	if len(parts) < 3 {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid state format, expected organizationID:userID:timestamp")
	}

	organizationID, err = uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	userID, err = uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return organizationID, userID, nil
}

func (s *slackConnector) CompleteAuthorization(authData backend.AuthorizationData) (backend.Credentials, error) {
	if authData.Code == "" {
		return backend.Credentials{}, fmt.Errorf("authorization code is required")
	}

	// Use slack-go library for OAuth2 token exchange
	oauthV2Response, err := slack.GetOAuthV2Response(
		s.client,
		s.config.ClientID,
		s.config.ClientSecret,
		authData.Code,
		s.config.RedirectURL,
	)
	if err != nil {
		return backend.Credentials{}, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	credentialData := map[string]string{
		"access_token":     oauthV2Response.AuthedUser.AccessToken,
		"bot_access_token": oauthV2Response.AccessToken,
		"bot_user_id":      oauthV2Response.BotUserID,
		"team_id":          oauthV2Response.Team.ID,
		"team_name":        oauthV2Response.Team.Name,
		"user_id":          oauthV2Response.AuthedUser.ID,
		"scope":            oauthV2Response.Scope,
	}

	organizationInfo := &backend.OrganizationInfo{
		ExternalID: oauthV2Response.Team.ID,
		Name:       oauthV2Response.Team.Name,
		Metadata: map[string]string{
			"connector_type": "slack",
		},
	}

	return backend.Credentials{
		Type:             backend.CredentialTypeOAuth2,
		Data:             credentialData,
		OrganizationInfo: organizationInfo,
	}, nil
}

func (s *slackConnector) ValidateCredentials(creds backend.Credentials) error {
	botToken, exists := creds.Data["bot_access_token"]
	if !exists {
		return fmt.Errorf("bot access token not found in credentials")
	}

	// Use slack-go library for auth test
	client := slack.New(botToken)
	authTest, err := client.AuthTest()
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	if authTest.UserID == "" {
		return fmt.Errorf("credential validation failed: invalid response")
	}

	return nil
}

func (s *slackConnector) RefreshCredentials(creds backend.Credentials) (backend.Credentials, error) {
	return creds, fmt.Errorf("Slack OAuth2 tokens do not support refresh")
}

func (s *slackConnector) RevokeCredentials(creds backend.Credentials) error {
	accessToken, exists := creds.Data["access_token"]
	if !exists {
		return fmt.Errorf("access token not found in credentials")
	}

	params := url.Values{}
	params.Set("token", accessToken)

	resp, err := s.client.PostForm("https://slack.com/api/auth.revoke", params)
	if err != nil {
		return fmt.Errorf("failed to revoke credentials: %w", err)
	}
	defer resp.Body.Close()

	var response struct {
		OK      bool   `json:"ok"`
		Revoked bool   `json:"revoked"`
		Error   string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode revoke response: %w", err)
	}

	if !response.OK {
		return fmt.Errorf("failed to revoke credentials: %s", response.Error)
	}

	return nil
}

func (s *slackConnector) ConfigureWebhooks(integrationID string, creds backend.Credentials) error {
	return nil
}

func (s *slackConnector) ValidateWebhookSignature(payload []byte, signature string, secret string) error {
	if secret == "" {
		secret = s.config.SigningSecret
	}

	expectedSignature := s.computeSignature(payload, secret)

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("webhook signature validation failed")
	}

	return nil
}

func (s *slackConnector) computeSignature(payload []byte, secret string) string {
	timestamp := time.Now().Unix()
	baseString := fmt.Sprintf("v0:%d:%s", timestamp, string(payload))

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(baseString))

	return fmt.Sprintf("v0=%s", hex.EncodeToString(h.Sum(nil)))
}

func (s *slackConnector) Subscribe(ctx context.Context, handler func(ctx context.Context, event any) error) error {
	if s.config.BotToken == "" {
		return fmt.Errorf("slack: bot token is required for Socket Mode")
	}
	if s.config.AppToken == "" {
		return fmt.Errorf("slack: app token is required for Socket Mode")
	}

	// TODO: Implement Socket Mode client when Slack library is available
	// For now, return a placeholder implementation
	//
	// Example implementation would be:
	// client := socketmode.New(
	//     slack.New(s.config.BotToken),
	//     socketmode.OptionAppToken(s.config.AppToken),
	// )
	//
	// go func() {
	//     for evt := range client.Events {
	//         switch evt.Type {
	//         case socketmode.EventTypeEventsAPI:
	//             eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	//             if !ok {
	//                 continue
	//             }
	//
	//             messageEvent := s.convertToMessageEvent(eventsAPIEvent)
	//             if err := handler(ctx, messageEvent); err != nil {
	//                 // Log error but continue processing
	//             }
	//         }
	//     }
	// }()
	//
	// return client.Run()

	return fmt.Errorf("slack Socket Mode implementation pending - requires slack-go library")
}

func (s *slackConnector) convertToMessageEvent(rawEvent any) MessageEvent {
	// TODO: Convert Slack Socket Mode events to our MessageEvent format
	// This would parse different event types (message, slash command, etc.)
	// and create appropriate MessageEvent structs

	return MessageEvent{
		EventType: EventTypeMessage,
		TeamID:    "",
		ChannelID: "",
		UserID:    "",
		Text:      "",
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
		CreatedAt: time.Now(),
		RawEvent:  make(map[string]any),
	}
}

func (s *slackConnector) ProcessEvent(ctx context.Context, event any) error {
	// Slack connector doesn't process events through this method
	// Events are handled directly in the conversation service
	// This is a no-op implementation
	return nil
}

func (s *slackConnector) Sync(ctx context.Context, integration backend.Integration, params map[string]string) error {
	// Sync workspace information and validate credentials
	if err := s.syncWorkspace(ctx, integration); err != nil {
		return fmt.Errorf("failed to sync workspace: %w", err)
	}

	// Sync channels information
	if err := s.syncChannels(ctx, integration); err != nil {
		return fmt.Errorf("failed to sync channels: %w", err)
	}

	return nil
}

func (s *slackConnector) syncWorkspace(ctx context.Context, integration backend.Integration) error {
	// TODO: Implement workspace synchronization
	// This could validate credentials and update workspace information
	return s.ValidateCredentials(backend.Credentials{
		Type: backend.CredentialTypeOAuth2,
		Data: map[string]string{
			"bot_access_token": integration.Metadata["bot_access_token"],
		},
	})
}

func (s *slackConnector) syncChannels(ctx context.Context, integration backend.Integration) error {
	// TODO: Implement channel synchronization
	// This could fetch and store channel information for the workspace
	// For now, this is a no-op
	return nil
}
