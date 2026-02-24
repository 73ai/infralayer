![InfraLayer](docs/assets/logo.svg)

![PyPI](https://img.shields.io/pypi/v/infralayer)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/73ai/infralayer/deploy.yml)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/73ai/infralayer/publish.yml)

InfraLayer is an AI SRE Copilot for the Cloud that provides infrastructure management agents through Slack integration. The system consists of multiple services that work together to deliver intelligent DevOps workflows.


<img src="docs/assets/slack-chat.png" style="max-width: 500px; height: auto;">

<img src="docs/assets//slack-cost-optimization-alert.png" style="max-width: 500px; height: auto;">

## WorkFlow

The services work together in this message flow:
1. User posts in Slack channel or uses CLI
2. Backend Service receives requests via Socket Mode
3. Backend Service calls Agent Service via gRPC for AI processing
4. Agent Service processes with LLM intelligence
5. Responses flow back through the system to Slack or CLI

## Features

- **🗣️ Natural Language Processing**: Convert natural language to infrastructure commands
- **🔗 Slack Integration**: Seamless Slack bot for team collaboration
- **🧠 Multi-Agent AI**: Intelligent routing and specialized agent responses
- **📊 Web Dashboard**: Modern web interface for platform management
- **🏗️ Infrastructure as Code**: Generate Terraform and other IaC
- **📈 Monitoring**: Track usage and infrastructure changes
- **🔐 Enterprise Security**: Authentication, authorization, and audit trails

## Integrations
We use a flexible data model so that we can support multiple integrations. Currently, InfraLayer supports Slack, GitHub and Terraform. 
We are actively working on adding integrations to the our stack.

## Platform Architecture

### 1. 🖥️ CLI Tool (`/cli/`)

Interactive terminal interface for infralayer

- Natural language queries for infrastructure commands
- Interactive mode with command history
- Support for OpenAI GPT-4o and Anthropic Claude models
- Install with: `pipx install infralayer`

![Demo CLI](docs/assets/infralayer.gif)

#### Quick Start

```bash
# Install using pipx (recommended)
pipx install infralayer

# Launch interactive mode
infralayer

# Example usage
> create a new VM instance called test-vm in us-central1 with 2 CPUs
```


[**📖 CLI Documentation**](cli/README.md)

### 2. Agent Service (`/services/agent/`)
Multi-agent framework with LLM integration

- Conversation management and RCA analysis
- FastAPI + gRPC dual server architecture
- Integration with Backend service

### 3. Backend Service (`/services/backend/`)

Main Slack bot and infrastructure management service

- Slack Socket Mode integration
- PostgreSQL-backed persistence
- GitHub PR management
- Terraform code generation
- Clean architecture with domain/infrastructure layers


```mermaid
  graph TD

  subgraph Core Services
    GoBackend[Go Backend<br>Service]
    PythonAgent[Python Agent<br>Service]
    GoBackend <--> |gRPC| PythonAgent
  end

  PythonAgent --> MainAgent["Main Agent<br>(Orchestrator)"]

  MainAgent --> ConversationAgent[Conversation<br>Agent]
  MainAgent --> RCAAagent[RCA<br>Agent]
  MainAgent --> FutureAgents[Future<br>Agents]

  ConversationAgent --> ToolManager
  RCAAagent --> ToolManager
  FutureAgents --> ToolManager

  ToolManager["Tool Manager<br>(Unified)"]

  ToolManager --> LangChain["LangChain<br>Tools"]
  ToolManager --> MCP["MCP Servers<br>(kubectl-ai,<br>gcloud, github)"]
  ToolManager --> CustomTools[Custom<br>Tools]
```

### 4. Console Service (`/services/console/`)
Web client interface for InfraLayer platform

- Modern React with Vite and TypeScript
- Radix UI components with Tailwind CSS
- Authentication via Clerk
- Real-time integration with platform services
## Contributing

For information on how to contribute to InfraLayer, including development setup, release process, and CI/CD configuration, please see the [CONTRIBUTING.md](CONTRIBUTING.md) file.

## License

This project is licensed under the GPL-3.0 License - see the [LICENSE](LICENSE) file for details.