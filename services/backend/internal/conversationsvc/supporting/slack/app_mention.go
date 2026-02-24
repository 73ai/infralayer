package slack

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func (s *Slack) handleAppMention(ctx context.Context, teamID string, event *slackevents.AppMentionEvent, handler func(context.Context, domain.UserCommand) error) error {
	teamToken, err := s.tokenRepository.GetToken(ctx, teamID)
	if err != nil {
		return fmt.Errorf("error getting team token for team_id:%s err:%w", teamID, err)
	}

	teamClient := slack.New(teamToken)

	channelInfo, err := teamClient.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID: event.Channel,
	})
	if err != nil {
		slog.Error("Error getting channel info", "error", err, "channelID", event.Channel)
	} else {
		err = s.channelRepository.AddChannel(ctx, teamID, event.Channel, channelInfo.Name)
		if err != nil {
			slog.Error("Error adding channel to DB", "error", err, "teamID", teamID, "channelID", event.Channel)
		}

		err = s.channelRepository.SetChannelMonitoring(ctx, teamID, event.Channel, true)
		if err != nil {
			slog.Error("Error setting channel monitoring", "error", err, "teamID", teamID, "channelID", event.Channel)
		} else {
			slog.Info("Successfully created and enabled monitoring for channel via app mention", "teamID", teamID, "channelID", event.Channel)
		}
	}

	at, err := teamClient.AuthTest()
	if err != nil {
		return fmt.Errorf("error authenticating team: %w", err)
	}

	botUserID := at.UserID

	// Extract text without the bot mention
	text := strings.TrimSpace(strings.Replace(event.Text, fmt.Sprintf("<@%s>", botUserID), "", -1))

	if err := teamClient.AddReaction("eyes", slack.NewRefToMessage(event.Channel, event.TimeStamp)); err != nil {
		slog.Error("Error adding reaction to app mention", "error", err, "channelID", event.Channel, "timestamp", event.TimeStamp)
	}

	// Get requester info
	requesterInfo, err := teamClient.GetUserInfo(event.User)
	requesterName := ""
	requesterUsername := ""
	requesterEmail := ""
	if err == nil && requesterInfo != nil {
		requesterName = requesterInfo.RealName
		requesterUsername = requesterInfo.Name // This is the @username
		requesterEmail = requesterInfo.Profile.Email
	} else {
		slog.Error("Error getting requester info:", "err", err)
	}

	var inReply bool
	var threadTimeStamp string
	// check if it is new thread or existing thread
	if event.ThreadTimeStamp == "" {
		inReply = false
		threadTimeStamp = event.TimeStamp
	} else {
		inReply = true
		threadTimeStamp = event.ThreadTimeStamp
	}

	m := domain.SlackThread{
		TeamID:   teamID,
		Channel:  event.Channel,
		ThreadTS: threadTimeStamp,
		Sender: domain.SlackUser{
			ID:       event.User,
			Email:    requesterEmail,
			Name:     requesterName,
			Username: requesterUsername,
		},
		Message: text,
	}

	command := domain.UserCommand{
		Thread:      m,
		InReply:     inReply,
		MessageType: domain.MessageTypeAppMention,
		MessageTS:   event.TimeStamp,
	}

	return handler(ctx, command)
}
