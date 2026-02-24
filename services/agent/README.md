# InfraLayer Agent Service

AI-powered infrastructure management agent providing intelligent processing for DevOps workflows via FastAPI + gRPC.

## Quick Start

```bash
uv sync --dev                     # Install dependencies
cp .env.example .env              # Configure environment
uv run python main.py            # Start service
uv run python -m pytest tests/ -v # Run tests
```

## Endpoints

- **HTTP (8000)**: `/health/`, `/health/ready`, `/health/live`
- **gRPC (50051)**: `ProcessMessage` for agent processing

## Agent System

- **Main Agent**: Orchestrates and routes requests
- **Conversation Agent**: Handles general user interactions
- **RCA Agent**: Performs root cause analysis for technical issues
- **LLM Integration**: LiteLLM client with intelligent routing

## Configuration

Set environment variables with `AGENT_` prefix:
- `AGENT_HOST`: Host to bind (default: `0.0.0.0`)
- `AGENT_HTTP_PORT`: FastAPI port (default: `8000`)
- `AGENT_GRPC_PORT`: gRPC port (default: `50051`)
- `AGENT_LOG_LEVEL`: Logging level (default: `INFO`)