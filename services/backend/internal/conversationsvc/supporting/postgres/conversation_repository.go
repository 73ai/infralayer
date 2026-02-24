package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
	"github.com/google/uuid"
)

func (db *BackendDB) GetConversationByThread(ctx context.Context, teamID, channelID, threadTS string) (domain.Conversation, error) {
	dbConversation, err := db.Querier.GetConversationByThread(ctx, GetConversationByThreadParams{
		TeamID:    teamID,
		ChannelID: channelID,
		ThreadTs:  threadTS,
	})
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("failed to get conversation: %w", err)
	}

	return domain.Conversation{
		ID:        dbConversation.ConversationID,
		TeamID:    dbConversation.TeamID,
		ChannelID: dbConversation.ChannelID,
		ThreadTS:  dbConversation.ThreadTs,
		CreatedAt: dbConversation.CreatedAt,
		UpdatedAt: dbConversation.UpdatedAt,
	}, nil
}

func (db *BackendDB) CreateConversation(ctx context.Context, teamID, channelID, threadTS string) (domain.Conversation, error) {
	dbConversation, err := db.Querier.CreateConversation(ctx, CreateConversationParams{
		TeamID:    teamID,
		ChannelID: channelID,
		ThreadTs:  threadTS,
	})
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("failed to create conversation: %w", err)
	}

	return domain.Conversation{
		ID:        dbConversation.ConversationID,
		TeamID:    dbConversation.TeamID,
		ChannelID: dbConversation.ChannelID,
		ThreadTS:  dbConversation.ThreadTs,
		CreatedAt: dbConversation.CreatedAt,
		UpdatedAt: dbConversation.UpdatedAt,
	}, nil
}

func (db *BackendDB) StoreMessage(ctx context.Context, conversationID uuid.UUID, message domain.Message) (domain.Message, error) {
	var senderUsername, senderEmail, senderName sql.NullString

	if message.Sender.Username != "" {
		senderUsername = sql.NullString{String: message.Sender.Username, Valid: true}
	}
	if message.Sender.Email != "" {
		senderEmail = sql.NullString{String: message.Sender.Email, Valid: true}
	}
	if message.Sender.Name != "" {
		senderName = sql.NullString{String: message.Sender.Name, Valid: true}
	}

	dbMessage, err := db.Querier.StoreMessage(ctx, StoreMessageParams{
		ConversationID: conversationID,
		SlackMessageTs: message.SlackMessageTS,
		SenderUserID:   message.Sender.ID,
		SenderUsername: senderUsername,
		SenderEmail:    senderEmail,
		SenderName:     senderName,
		MessageText:    message.MessageText,
		IsBotMessage:   message.IsBotMessage,
	})
	if err != nil {
		return domain.Message{}, fmt.Errorf("failed to store message: %w", err)
	}

	return domain.Message{
		ID:             dbMessage.MessageID,
		ConversationID: dbMessage.ConversationID,
		SlackMessageTS: dbMessage.SlackMessageTs,
		Sender: domain.SlackUser{
			ID:       dbMessage.SenderUserID,
			Username: dbMessage.SenderUsername.String,
			Email:    dbMessage.SenderEmail.String,
			Name:     dbMessage.SenderName.String,
		},
		MessageText:  dbMessage.MessageText,
		IsBotMessage: dbMessage.IsBotMessage,
		CreatedAt:    dbMessage.CreatedAt,
	}, nil
}

func (db *BackendDB) GetConversationHistory(ctx context.Context, conversationID uuid.UUID) ([]domain.Message, error) {
	dbMessages, err := db.Querier.GetConversationHistory(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	messages := make([]domain.Message, len(dbMessages))
	for i, dbMsg := range dbMessages {
		messages[i] = domain.Message{
			ID:             dbMsg.MessageID,
			ConversationID: dbMsg.ConversationID,
			SlackMessageTS: dbMsg.SlackMessageTs,
			Sender: domain.SlackUser{
				ID:       dbMsg.SenderUserID,
				Username: dbMsg.SenderUsername.String,
				Email:    dbMsg.SenderEmail.String,
				Name:     dbMsg.SenderName.String,
			},
			MessageText:  dbMsg.MessageText,
			IsBotMessage: dbMsg.IsBotMessage,
			CreatedAt:    dbMsg.CreatedAt,
		}
	}

	return messages, nil
}

func (db *BackendDB) MessageBySlackTS(ctx context.Context, conversationID uuid.UUID, senderID, slackMessageTS string) (domain.Message, error) {
	dbMessage, err := db.Querier.MessageBySlackTS(ctx, MessageBySlackTSParams{
		ConversationID: conversationID,
		SenderUserID:   senderID,
		SlackMessageTs: slackMessageTS,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Message{}, sql.ErrNoRows
		}
		return domain.Message{}, fmt.Errorf("failed to get message by slack timestamp: %w", err)
	}

	return domain.Message{
		ID:             dbMessage.MessageID,
		ConversationID: dbMessage.ConversationID,
		SlackMessageTS: dbMessage.SlackMessageTs,
		Sender: domain.SlackUser{
			ID:       dbMessage.SenderUserID,
			Username: dbMessage.SenderUsername.String,
			Email:    dbMessage.SenderEmail.String,
			Name:     dbMessage.SenderName.String,
		},
		MessageText:  dbMessage.MessageText,
		IsBotMessage: dbMessage.IsBotMessage,
		CreatedAt:    dbMessage.CreatedAt,
	}, nil
}

func (db *BackendDB) Conversation(ctx context.Context, conversationID uuid.UUID) (domain.Conversation, error) {
	dbConversation, err := db.Querier.Conversation(ctx, conversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Conversation{}, sql.ErrNoRows
		}
		return domain.Conversation{}, fmt.Errorf("failed to get conversation: %w", err)
	}

	return domain.Conversation{
		ID:        dbConversation.ConversationID,
		TeamID:    dbConversation.TeamID,
		ChannelID: dbConversation.ChannelID,
		ThreadTS:  dbConversation.ThreadTs,
		CreatedAt: dbConversation.CreatedAt,
		UpdatedAt: dbConversation.UpdatedAt,
	}, nil
}

var _ domain.ConversationRepository = (*BackendDB)(nil)
