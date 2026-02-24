package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	agent "github.com/73ai/infralayer/services/agent/src/client/go"
	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
)

// Client wraps the agent gRPC client to implement domain.AgentService
type Client struct {
	agentClient *agent.Client
}

// NewClient creates a new agent client that connects to the Python agent service
func NewClient(config *agent.Config) (*Client, error) {
	client, err := agent.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent client: %w", err)
	}

	return &Client{
		agentClient: client,
	}, nil
}

// ProcessMessage implements domain.AgentService interface
func (c *Client) ProcessMessage(ctx context.Context, request domain.AgentRequest) (domain.AgentResponse, error) {
	// Convert domain models to agent protobuf models
	agentReq, err := c.convertToAgentRequest(request)
	if err != nil {
		return domain.AgentResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to convert request: %v", err),
		}, nil
	}

	// Call the Python agent service
	resp, err := c.agentClient.ProcessMessage(ctx, agentReq)
	if err != nil {
		log.Printf("Agent service error: %v", err)
		return domain.AgentResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("agent service unavailable: %v", err),
		}, nil
	}

	// Convert response back to domain model
	return domain.AgentResponse{
		ResponseText: resp.ResponseText,
		Success:      resp.Success,
		ErrorMessage: resp.ErrorMessage,
	}, nil
}

// Close closes the connection to the agent service
func (c *Client) Close() error {
	if c.agentClient != nil {
		return c.agentClient.Close()
	}
	return nil
}

// convertToAgentRequest converts domain.AgentRequest to agent client request
func (c *Client) convertToAgentRequest(req domain.AgentRequest) (agent.AgentRequest, error) {
	// Convert past messages to Message objects
	var pastMessages []*agent.Message
	for _, msg := range req.PastMessages {
		// Determine sender format: "agent" for bot messages, "user/{user_id}" for users
		sender := fmt.Sprintf("user/%s", msg.Sender.ID)
		if msg.IsBotMessage {
			sender = "agent"
		}

		pastMessages = append(pastMessages, &agent.Message{
			MessageId: msg.ID.String(),
			Content:   msg.MessageText,
			Sender:    sender,
			Timestamp: msg.CreatedAt.Format(time.RFC3339),
		})
	}

	return agent.AgentRequest{
		ConversationId: req.Message.ConversationID.String(),
		CurrentMessage: req.Message.MessageText,
		PastMessages:   pastMessages,
		UserId:         req.Message.Sender.Name,
		ChannelId:      req.Conversation.ChannelID,
	}, nil
}
