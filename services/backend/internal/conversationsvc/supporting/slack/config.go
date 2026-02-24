package slack

import (
	"context"
	"fmt"

	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type Config struct {
	ClientID                 string                          `mapstructure:"client_id"`
	ClientSecret             string                          `mapstructure:"client_secret"`
	AppToken                 string                          `mapstructure:"app_token"`
	WorkSpaceTokenRepository domain.WorkSpaceTokenRepository `mapstructure:"-"`
	ChannelRepository        domain.ChannelRepository        `mapstructure:"-"`
}

func (c Config) New(ctx context.Context) (*Slack, error) {
	if c.WorkSpaceTokenRepository == nil {
		return nil, fmt.Errorf("work space token repository is required")
	}
	if c.ChannelRepository == nil {
		return nil, fmt.Errorf("channel repository is required")
	}
	client := slack.New("", slack.OptionAppLevelToken(c.AppToken))
	socketClient := socketmode.New(client)

	if c.ClientID == "" {
		return nil, fmt.Errorf("client id is required")
	}
	if c.ClientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	return &Slack{
		clientID:          c.ClientID,
		clientSecret:      c.ClientSecret,
		client:            client,
		socketClient:      socketClient,
		tokenRepository:   c.WorkSpaceTokenRepository,
		channelRepository: c.ChannelRepository,
	}, nil
}
