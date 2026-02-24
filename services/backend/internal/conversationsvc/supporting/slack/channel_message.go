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

func (s *Slack) handleChannelMessage(ctx context.Context, teamID string, event *slackevents.MessageEvent, handler func(context.Context, domain.UserCommand) error) error {
	slog.Info("Handling channel message event", "teamID", teamID, "channelID", event.Channel, "user", event.User, "text", event.Text, "bot", event.BotID, "subType", event.SubType, "threadTS", event.ThreadTimeStamp,
		"e", event)
	// NOTE: This is a workaround for the bot user ID that is used in testing datadog bot.
	if event.BotID != "B090TCWJFDW" {
		if event.BotID != "" {
			return nil
		}

		if event.SubType != "" {
			return nil
		}
	}

	teamToken, err := s.tokenRepository.GetToken(ctx, teamID)
	if err != nil {
		return fmt.Errorf("error getting team token for team_id:%s err:%w", teamID, err)
	}

	teamClient := slack.New(teamToken)

	isMonitored, err := s.channelRepository.IsChannelMonitored(ctx, teamID, event.Channel)
	if err != nil {
		slog.Error("Error checking if channel is monitored, will create channel", "error", err, "teamID", teamID, "channelID", event.Channel)
	}

	if !isMonitored {
		return nil
	}

	channelInfo, err := teamClient.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID: event.Channel,
	})
	if err != nil {
		slog.Error("Error getting channel info", "error", err, "channelID", event.Channel)
		return nil
	}

	err = s.channelRepository.AddChannel(ctx, teamID, event.Channel, channelInfo.Name)
	if err != nil {
		slog.Error("Error adding channel to DB", "error", err, "teamID", teamID, "channelID", event.Channel)
		return nil
	}

	err = s.channelRepository.SetChannelMonitoring(ctx, teamID, event.Channel, true)
	if err != nil {
		slog.Error("Error setting channel monitoring", "error", err, "teamID", teamID, "channelID", event.Channel)
		return nil
	}

	isMonitored = true

	at, err := teamClient.AuthTest()
	if err != nil {
		return fmt.Errorf("error authenticating team: %w", err)
	}

	botUserID := at.UserID

	// check if it's mention to the bot
	if strings.Contains(event.Text, fmt.Sprintf("<@%s>", botUserID)) {
		slog.Info("Ignoring message mentioning the bot", "teamID", teamID, "channelID", event.Channel, "text", event.Text)
		return nil
	}

	// Extract text without the bot mention
	text := strings.TrimSpace(strings.Replace(event.Text, fmt.Sprintf("<@%s>", botUserID), "", -1))

	// Include attachment information if present
	if len(event.Attachments) > 0 {
		var b strings.Builder
		if text != "" {
			b.WriteString(text)
		}
		for _, attachment := range event.Attachments {
			title := strings.TrimSpace(attachment.Title)
			body := strings.TrimSpace(attachment.Text)
			if title == "" && body == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			// Title + optional link
			if title != "" {
				b.WriteString(title)
				if attachment.TitleLink != "" {
					b.WriteString(" (")
					b.WriteString(attachment.TitleLink)
					b.WriteByte(')')
				} else if attachment.FromURL != "" {
					b.WriteString(" (")
					b.WriteString(attachment.FromURL)
					b.WriteByte(')')
				}
			}
			// Separator + body
			if body != "" {
				if title != "" {
					b.WriteString(": ")
				}
				b.WriteString(body)
			}
		}
		text = strings.TrimSpace(b.String())
	}

	if err := teamClient.AddReaction("eyes", slack.NewRefToMessage(event.Channel, event.TimeStamp)); err != nil {
		slog.Error("Error adding reaction to app mention", "error", err, "channelID", event.Channel, "timestamp", event.TimeStamp)
	}

	requesterInfo, err := teamClient.GetUserInfo(event.User)
	requesterName := ""
	requesterUsername := ""
	requesterEmail := ""
	if err == nil && requesterInfo != nil {
		requesterName = requesterInfo.RealName
		requesterUsername = requesterInfo.Name
		requesterEmail = requesterInfo.Profile.Email
	} else {
		slog.Error("Error getting requester info:", "err", err)
	}

	var inReply bool
	var threadTimeStamp string
	var messageType domain.MessageType

	if event.ThreadTimeStamp == "" {
		inReply = false
		threadTimeStamp = event.TimeStamp
		messageType = domain.MessageTypeChannel
	} else {
		inReply = true
		threadTimeStamp = event.ThreadTimeStamp
		messageType = domain.MessageTypeThread
	}

	slackThread := domain.SlackThread{
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
		Thread:      slackThread,
		InReply:     inReply,
		MessageType: messageType,
		MessageTS:   event.TimeStamp,
	}

	return handler(ctx, command)
}
