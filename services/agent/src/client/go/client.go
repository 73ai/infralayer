package agent

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/73ai/infralayer/services/agent/src/client/go/proto"
)

// Message is an alias for the protobuf Message type for convenience
type Message = pb.Message

// Client provides a client interface to the agent gRPC service
type Client struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	config *Config
}

// Config holds configuration for the agent client
type Config struct {
	Endpoint       string        `mapstructure:"endpoint"`
	Timeout        time.Duration `mapstructure:"-"`
	RetryAttempts  int           `mapstructure:"retry_attempts"`
	ConnectTimeout time.Duration `mapstructure:"-"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Endpoint:       "localhost:50052",
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
		ConnectTimeout: 10 * time.Second,
	}
}

// NewClient creates a new agent client with the given configuration
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Set up connection with keepalive
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                5 * time.Minute,
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.Endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent service at %s: %w", config.Endpoint, err)
	}

	client := pb.NewAgentServiceClient(conn)

	return &Client{
		conn:   conn,
		client: client,
		config: config,
	}, nil
}

type AgentRequest struct {
	ConversationId string
	CurrentMessage string
	PastMessages   []*pb.Message
	Context        string
	UserId         string
	ChannelId      string
}

type AgentResponse struct {
	Success      bool
	ResponseText string
	ErrorMessage string
	AgentType    string
	Confidence   float32
	ToolsUsed    []string
}

// ProcessMessage sends a message to the agent for processing
func (c *Client) ProcessMessage(ctx context.Context, req AgentRequest) (AgentResponse, error) {
	// Create the gRPC request
	pbReq := &pb.AgentRequest{
		ConversationId: req.ConversationId,
		CurrentMessage: req.CurrentMessage,
		PastMessages:   req.PastMessages,
		Context:        req.Context,
		UserId:         req.UserId,
		ChannelId:      req.ChannelId,
	}

	// Set timeout for the request
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt < c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Linear backoff for retries
			delay := time.Duration(attempt) * time.Second
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return AgentResponse{}, ctx.Err()
			}
		}

		resp, err := c.client.ProcessMessage(ctx, pbReq)
		if err == nil {
			return AgentResponse{
				Success:      resp.Success,
				ResponseText: resp.ResponseText,
				ErrorMessage: resp.ErrorMessage,
				AgentType:    resp.AgentType,
				Confidence:   resp.Confidence,
				ToolsUsed:    resp.ToolsUsed,
			}, nil
		}

		lastErr = err

		// Don't retry on context cancellation/timeout
		if ctx.Err() != nil {
			break
		}
	}

	return AgentResponse{}, fmt.Errorf("failed to process message after %d attempts: %w", c.config.RetryAttempts, lastErr)
}

// Close closes the connection to the agent service
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
