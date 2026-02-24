# Agent Go Client

Go client library for the Backend Agent gRPC service.

## Usage

```go
package main

import (
    "context"
    "log"
    
    agent "github.com/73ai/infralayer/services/agent/client/go"
)

func main() {
    // Create client with default config
    client, err := agent.NewClient(nil)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Process a message
    req := &agent.AgentRequest{
        ConversationId: "conv-123",
        CurrentMessage: "Help me troubleshoot a failed deployment",
        UserId:        "user-456",
        ChannelId:     "channel-789",
    }
    
    resp, err := client.ProcessMessage(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }
    
    if resp.Success {
        log.Printf("Agent response: %s", resp.ResponseText)
    } else {
        log.Printf("Agent error: %s", resp.ErrorMessage)
    }
}
```

## Configuration

```go
config := &agent.Config{
    Endpoint:       "localhost:50052",
    Timeout:        30 * time.Second,
    RetryAttempts:  3,
    ConnectTimeout: 10 * time.Second,
}

client, err := agent.NewClient(config)
```

## Health Check

```go
if client.IsHealthy(context.Background()) {
    log.Println("Agent service is healthy")
}
```