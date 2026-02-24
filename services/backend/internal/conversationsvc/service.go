package conversationsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/73ai/infralayer/services/backend"
	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
	"github.com/google/uuid"
)

type Service struct {
	slackGateway           domain.SlackGateway
	integrationRepository  domain.IntegrationRepository
	conversationRepository domain.ConversationRepository
	channelRepository      domain.ChannelRepository
	agentService           domain.AgentService
}

func (s *Service) Integrations(ctx context.Context, query backend.IntegrationsQuery) ([]backend.Integration, error) {
	is, err := s.integrationRepository.Integrations(ctx, query.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get integrations: %w", err)
	}

	var integrations []backend.Integration
	for _, i := range is {
		integrations = append(integrations, backend.Integration{
			ConnectorType: i.ConnectorType,
			Status:        i.Status,
		})
	}

	return integrations, nil
}

func (s *Service) CompleteSlackIntegration(ctx context.Context, command backend.CompleteSlackIntegrationCommand) error {
	if pid, err := s.slackGateway.CompleteAuthentication(ctx, command.Code); err != nil {
		return fmt.Errorf("failed to complete slack authentication: %w", err)
	} else {
		err := s.integrationRepository.SaveIntegration(ctx, domain.Integration{
			Integration: backend.Integration{
				ConnectorType: backend.ConnectorTypeSlack,
				Status:        backend.IntegrationStatusActive,
			},
			BusinessID:        command.BusinessID,
			ProviderProjectID: pid,
		})
		if err != nil {
			return fmt.Errorf("failed to complete slack authentication: %w", err)
		}
	}

	return nil
}

var _ backend.ConversationService = (*Service)(nil)

func (s *Service) SendReply(ctx context.Context, command backend.SendReplyCommand) error {
	slog.Info("Sending reply to Slack", "conversationID", command.ConversationID, "message", command.Message)
	conversationID, err := uuid.Parse(command.ConversationID)
	if err != nil {
		return fmt.Errorf("invalid conversation ID: %w", err)
	}

	conversation, err := s.conversationRepository.Conversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	thread := domain.SlackThread{
		Message:  "",
		Channel:  conversation.ChannelID,
		ThreadTS: conversation.ThreadTS,
		TeamID:   conversation.TeamID,
	}

	err = s.slackGateway.ReplyMessage(ctx, thread, command.Message)
	if err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}

	botMessage := domain.Message{
		ConversationID: conversationID,
		SlackMessageTS: fmt.Sprintf("%d", time.Now().UnixNano()),
		Sender: domain.SlackUser{
			ID:       "bot",
			Username: "bot",
			Name:     "Backend Bot",
		},
		MessageText:  command.Message,
		IsBotMessage: true,
	}

	_, err = s.conversationRepository.StoreMessage(ctx, conversationID, botMessage)
	if err != nil {
		slog.Error("Failed to store bot message", "error", err)
		return fmt.Errorf("failed to store bot message: %w", err)
	}

	return nil
}

func (s *Service) SubscribeSlackNotifications(ctx context.Context) error {
	if err := s.slackGateway.SubscribeAllMessages(ctx, s.handleUserCommand); err != nil {
		return fmt.Errorf("failed to subscribe to all messages: %w", err)
	}

	return nil
}

func (s *Service) handleUserCommand(ctx context.Context, command domain.UserCommand) error {
	slog.Info("Received user command", "type", command.MessageType, "channel", command.Thread.Channel, "user", command.Thread.Sender.Username)

	var pastMessages []domain.Message

	var conversation domain.Conversation
	var err error
	conversation, err = s.conversationRepository.GetConversationByThread(ctx, command.Thread.TeamID, command.Thread.Channel, command.Thread.ThreadTS)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		slog.Error("Failed to get conversation", "error", err)
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		conversation, err = s.conversationRepository.CreateConversation(ctx, command.Thread.TeamID, command.Thread.Channel, command.Thread.ThreadTS)
		if err != nil {
			slog.Error("Failed to create conversation", "error", err)
			return fmt.Errorf("failed to create conversation: %w", err)
		}
	} else {
		pastMessages, err = s.conversationRepository.GetConversationHistory(ctx, conversation.ID)
		if err != nil {
			slog.Error("Failed to get conversation history", "error", err)
			return fmt.Errorf("failed to get conversation history: %w", err)
		}
	}

	message := domain.Message{
		ConversationID: conversation.ID,
		SlackMessageTS: fmt.Sprintf("%d", time.Now().UnixNano()),
		Sender:         command.Thread.Sender,
		MessageText:    command.Thread.Message,
		IsBotMessage:   false,
	}

	_, err = s.conversationRepository.MessageBySlackTS(ctx, conversation.ID, command.Thread.Sender.ID, command.MessageTS)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Info("No existing message found, proceeding to store new message")
		} else {
			slog.Error("Failed to get messages by Slack timestamp", "error", err)
			return fmt.Errorf("failed to get messages by Slack timestamp: %w", err)
		}
	}

	_, err = s.conversationRepository.StoreMessage(ctx, conversation.ID, message)
	if err != nil {
		slog.Error("Failed to store message", "error", err)
		return fmt.Errorf("failed to store message: %w", err)
	}

	agentRequest := domain.AgentRequest{
		Conversation: conversation,
		Message:      message,
		PastMessages: pastMessages,
	}

	_, err = s.agentService.ProcessMessage(ctx, agentRequest)
	if err != nil {
		slog.Error("Failed to process message with agent service", "error", err)
	}

	return nil
}
