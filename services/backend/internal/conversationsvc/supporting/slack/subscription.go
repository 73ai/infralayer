package slack

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func (s *Slack) subscribe(ctx context.Context, handler func(context.Context, domain.UserCommand) error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-s.socketClient.Events:
			slog.Info("Received event from Slack", "type", event.Type)
			switch event.Type {
			case socketmode.EventTypeConnecting:
				slog.Info("Connecting to Slack API...")
			case socketmode.EventTypeConnectionError:
				slog.Info("Connection error:", "data", event.Data)
			case socketmode.EventTypeConnected:
				slog.Info("Connected to Slack!")
			case socketmode.EventTypeEventsAPI:
				s.socketClient.Ack(*event.Request)
				payload, ok := event.Data.(slackevents.EventsAPIEvent)
				if !ok {
					slog.Error("Failed to cast event data to EventsAPIEvent", "msg", event.Data)
					continue
				}
				err := s.handleEventAPI(ctx, payload, handler)
				if err != nil {
					slog.Error("Failed to handle event API:", "error", err)
				}
			default:
				slog.Info("Unhandled event type: %s with data:",
					"type", event.Type, "data", event.Data)
			}
		}
	}
}

func (s *Slack) handleEventAPI(ctx context.Context, event slackevents.EventsAPIEvent, handler func(context.Context, domain.UserCommand) error) error {
	teamID := event.TeamID
	switch event.Type {
	case slackevents.CallbackEvent:
		switch ev := event.InnerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			err := s.handleAppMention(ctx, teamID, ev, handler)
			if err != nil {
				return fmt.Errorf("failed to handle app mention: %w", err)
			}
		case *slackevents.MessageEvent:
			err := s.handleChannelMessage(ctx, teamID, ev, handler)
			if err != nil {
				return fmt.Errorf("failed to handle channel message: %w", err)
			}
		default:
			slog.Info("Unhandled callback event:", "event", ev)
		}
	case slackevents.URLVerification:
		slog.Info("Received URL verification event")
	default:
		slog.Info("Unhandled event", "type", event.Type, "data", event.InnerEvent.Data)
	}

	return nil
}
