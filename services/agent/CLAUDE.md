# Agent Service

This service provides intelligent AI processing capabilities for DevOps workflows within the InfraLayer platform.

## Service Purpose

The Agent Service is a Python-based AI processing component that serves as the intelligent brain of the InfraLayer platform. It handles natural language understanding, workflow orchestration, and contextual responses for infrastructure-related tasks. The service communicates with the main Go service via gRPC and provides AI-powered responses through Slack interactions.

## Architecture

### Multi-Agent Framework
- **Main Agent**: Intelligent routing and workflow orchestration using LLM-based intent detection
- **Conversation Agent**: LLM-powered contextual dialogue for infrastructure topics
- **RCA Agent**: LLM-driven root cause analysis with structured technical recommendations
- **Agent Registry**: Centralized agent management and routing system

### Core Components
- **Dual Server**: FastAPI (health endpoints) + gRPC (agent processing)
- **LLM Integration**: LiteLLM client with conversation management and shared resource pooling
- **Tool Framework**: Extensible tool registry ready for MCP integration
- **Configuration**: Pydantic-based settings with environment variable support

## Development Commands

```bash
# Run the agent service
uv run python main.py

# Install dependencies
uv sync

# Run tests
uv run pytest tests/ -v

# Run specific test module
uv run pytest tests/test_agents.py -v
```

## Project Structure

```
/services/agent/
├── src/
│   ├── config/          # Pydantic settings and logging
│   ├── api/             # FastAPI health endpoints
│   ├── grpc/            # gRPC server and handlers
│   ├── agents/          # Multi-agent implementations
│   ├── llm/             # LiteLLM client integration
│   ├── tools/           # Tool registry framework
│   ├── models/          # Data models and types
│   └── integrations/    # Backend API client wrapper
├── tests/               # Test suite
├── main.py             # Service entry point
└── Dockerfile          # Container configuration
```

## LLM Integration Patterns

### Client Management
```python
# Shared LLM client across all agents
from src.llm.client import LLMClient
client = LLMClient(model="gpt-4", api_key=settings.api_key)
```

### Conversation Context
```python
# Multi-turn conversation support
response = await client.generate_response(
    messages=conversation_history,
    context={"intent": "troubleshooting", "domain": "infrastructure"}
)
```

### Agent Routing
```python
# LLM-powered intent detection for agent selection
intent = await llm_client.analyze_intent(user_message)
agent = agent_registry.get_agent_for_intent(intent)
```

## gRPC Integration

### Message Flow
1. Slack message → Go service (Socket Mode)
2. Go service → Python agent (gRPC ProcessMessage)
3. Agent processes with LLM intelligence
4. Agent replies via Backend API client
5. Response appears in Slack thread

### Service Interface
```python
async def ProcessMessage(self, request: AgentRequest) -> AgentResponse:
    agent = self.agent_registry.get_main_agent()
    response = await agent.process_message(request.message, request.context)
    await self.reply_handler.send_agent_response(response)
    return AgentResponse(success=True)
```

## Agent Capabilities

### Conversation Agent
- Natural language understanding for infrastructure topics
- Contextual dialogue management
- Professional interaction patterns

### RCA Agent  
- Intelligent issue analysis and categorization
- Structured troubleshooting recommendations
- Pattern recognition for error types

### Tool Integration
- Base tool classes with execution framework
- Registry-based tool discovery and management
- Ready for MCP (Model Context Protocol) integration

## Testing and Deployment

### Test Structure
- Unit tests for individual agents and components
- Integration tests for gRPC communication
- LLM interaction mocks for reliable testing

### Configuration
```yaml
# Agent service config in Go service
agent:
  endpoint: "localhost:50052"
  timeout: "30s"
  retry_attempts: 3
```

### Health Monitoring
- `/health` - Basic service health
- `/ready` - Service readiness with dependency checks
- `/live` - Liveness probe for container orchestration

## Related Components

- Main Backend service: `../backend/`
- gRPC client: `../infralayer/infralayerapi/client/python/`
- Go domain models: `../infralayer/internal/infralayersvc/domain/agent.go`