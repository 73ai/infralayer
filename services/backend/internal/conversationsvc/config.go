package conversationsvc

import (
	"context"
	"fmt"

	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
)

type Config struct {
	SlackGateway           domain.SlackGateway
	IntegrationRepository  domain.IntegrationRepository
	ConversationRepository domain.ConversationRepository
	ChannelRepository      domain.ChannelRepository
	AgentService           domain.AgentService
}

func (c Config) New(ctx context.Context) (*Service, error) {
	if c.SlackGateway == nil {
		return nil, fmt.Errorf("slack gateway is required")
	}
	if c.IntegrationRepository == nil {
		return nil, fmt.Errorf("integration repository is required")
	}
	if c.ConversationRepository == nil {
		return nil, fmt.Errorf("conversation repository is required")
	}
	if c.ChannelRepository == nil {
		return nil, fmt.Errorf("channel repository is required")
	}
	if c.AgentService == nil {
		return nil, fmt.Errorf("agent service is required")
	}
	return &Service{
		slackGateway:           c.SlackGateway,
		integrationRepository:  c.IntegrationRepository,
		conversationRepository: c.ConversationRepository,
		channelRepository:      c.ChannelRepository,
		agentService:           c.AgentService,
	}, nil
}
