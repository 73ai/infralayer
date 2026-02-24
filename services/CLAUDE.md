# Services Overview

InfraLayer is a multi-service platform providing AI-powered infrastructure management through Slack integration. The system consists of three main services that work together to deliver intelligent DevOps workflows.

## Core Services

### Backend Service
**Purpose**: Central Slack bot and infrastructure management service that handles user interactions, maintains conversation state, and coordinates workflow execution.

**Role**: Acts as the primary interface for users interacting with the platform through Slack. Manages authentication, organization context, and serves as the orchestration layer for all platform operations.

### Agent Service  
**Purpose**: AI-powered message processing engine that interprets natural language requests and generates intelligent responses for infrastructure management tasks.

**Role**: Provides the cognitive layer of the platform, processing user intents and converting them into actionable infrastructure commands and recommendations.

### Console Service
**Purpose**: Administrative interface for platform management, configuration, and monitoring of infrastructure operations.

**Role**: Enables users to manage platform settings, view operation history, configure integrations, and access detailed analytics outside of the Slack interface.

## Architecture and Integration Flow

The services form a cohesive ecosystem where each component has a specific responsibility:

1. **User Interaction Layer**: Users interact primarily through Slack channels, with the Console Service providing administrative capabilities
2. **Message Processing**: The Backend Service receives all user messages and determines routing and context
3. **AI Processing**: The Agent Service applies natural language understanding and generates intelligent responses
4. **Response Delivery**: Processed responses flow back through the Backend Service to users via Slack
5. **Management Interface**: The Console Service provides oversight and configuration capabilities across all operations

## Service Communication Patterns

### Synchronous Communication
- Backend Service communicates with Agent Service via gRPC for real-time message processing
- Console Service communicates with Backend Service via REST APIs for configuration management

### Shared Context
- All services maintain consistent authentication and organization context
- Database state is managed centrally through the Backend Service
- Configuration changes propagate across services through established patterns

### Event-Driven Operations
- Slack events trigger the primary workflow through Socket Mode integration
- Service health and operational metrics flow through monitoring channels

## Development Approach

### Service Independence
Each service is designed to be independently deployable and maintainable while participating in the larger ecosystem.

### Shared Standards
- Go services follow standard Go conventions and patterns
- Python services use modern async patterns and type hints
- TypeScript/React services follow current best practices

### Testing Strategy
Each service maintains its own comprehensive test suite, with integration testing ensuring proper service communication.

## Navigation

For detailed implementation information, configuration specifics, and development guidelines, refer to the individual service documentation within each service directory.