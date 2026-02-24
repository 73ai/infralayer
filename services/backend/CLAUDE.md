# Backend Service

The main backend service that orchestrates DevOps workflows through Slack integration and serves as the platform's communication hub.

## Service Purpose

The Backend Service is the central backend service built with Go that coordinates all platform operations. It provides real-time Slack integration, user authentication, workflow orchestration, and serves as the communication bridge between Slack, the Agent Service (AI processing), and the Web Application.

**Core Responsibilities:**
- Slack Socket Mode integration for real-time messaging
- User authentication and session management
- DevOps access request workflow orchestration
- Integration management with external services
- gRPC communication with Agent Service for AI processing

## Architecture

### Clean Architecture Layers
The service follows clean architecture principles with dependency inversion:

- **Domain Layer** (`internal/conversationsvc/domain/`) - Core business logic, entities, and repository interfaces
- **Application Layer** (`internal/conversationsvc/service.go`) - Service orchestration, business workflows, and use cases
- **Infrastructure Layer** (`internal/conversationsvc/supporting/`) - External integrations (Slack, PostgreSQL, gRPC)
- **API Layer** (`backendapi/`) - HTTP endpoint handlers and request/response models

### Dual Service Design
- **Backend Service** (`internal/conversationsvc/`) - Main Slack bot, integration management, workflow orchestration
- **Identity Service** (`internal/identitysvc/`) - User authentication, session management, organization context

### Core Components
- **Main Entry Point**: `cmd/main.go` - Application bootstrapping, dependency injection, service initialization
- **Service Contracts**: `spec.go` - Main service interface definitions and method signatures
- **Database Layer**: PostgreSQL with SQLC for compile-time type-safe query generation
- **Slack Integration**: Socket Mode for real-time bidirectional communication
- **Configuration**: YAML-based config with mapstructure for structured parsing
- **gRPC Client**: Communication with Agent Service for AI-powered message processing

## Development Workflow

### Build and Run Commands
- `find . -name "*.go" -not -path "./vendor/*" -exec goimports -w {} \;` - Format code and organize imports (mandatory before commits)
- `go run ./cmd/main.go` - Start the backend service (requires config.yaml)
- `go build ./cmd/main.go` - Build production binary
- `go build -o infralayer ./cmd/main.go` - Build with custom binary name

### Testing and Quality Assurance
- `go test ./...` - Run complete test suite
- `go test ./internal/conversationsvc/... -v` - Verbose tests for Backend service
- `go test ./internal/identitysvc/... -v` - Verbose tests for Identity service
- `go vet ./...` - Static analysis for common Go errors
- `go mod verify` - Verify dependency integrity

### Database Development
- `sqlc generate` - Generate type-safe Go code from SQL (via sqlc.json configuration)
- `go test ./internal/*/supporting/postgres/...` - Run database integration tests
- Database migration handling through schema files

### Dependency Management
- `go mod tidy` - Clean up and optimize module dependencies
- `go mod download` - Download all dependencies
- `go mod graph` - Display dependency graph for debugging

## Configuration Management

### Environment-Specific Configurations
- Development: `config.yaml` with local PostgreSQL and test Slack workspace
- Production: Environment variables override config.yaml values

## Database Architecture and Management

### Multi-Service Database Pattern
- **Backend Service DB**: `internal/conversationsvc/supporting/postgres/`
  - Schema: `schema/schema.sql` - Integration management, workflow state
  - Queries: `queries/*.sql` - CRUD operations for integrations and workflows
  - Generated: `*.sql.go` - Type-safe query methods

- **Identity Service DB**: `internal/identitysvc/supporting/postgres/`  
  - Schema: `schema/schema.sql` - User authentication, sessions, organizations
  - Queries: `queries/*.sql` - User management and auth operations
  - Generated: `*.sql.go` - Type-safe authentication queries

### Database Development Patterns
- **SQLC Integration**: Write raw SQL, generate type-safe Go code
- **Repository Pattern**: Domain interfaces implemented by infrastructure layer
- **Migration Strategy**: Schema files managed manually with versioning
- **Constraint Design**: Single integration per organization/connector type
- **Indexing Strategy**: Optimized indexes for all query patterns

## Integration Patterns and Service Communication

### gRPC Integration with Agent Service
- **Client Configuration**: gRPC client setup in infrastructure layer
- **Message Processing Flow**: Slack message → Backend Service → Agent Service → LLM → Response
- **Error Handling**: Graceful degradation when Agent Service unavailable
- **Retry Logic**: Exponential backoff for failed gRPC calls

### Slack Socket Mode Integration
- **Real-time Events**: Bidirectional WebSocket connection with Slack
- **Event Types**: Message events, app mentions, reaction events
- **Authentication**: Bot token and app token validation
- **Rate Limiting**: Built-in Slack SDK rate limiting compliance

## Code Standards and Best Practices

### Go Code Style Guidelines
- Follow standard Go conventions from Effective Go
- Use gofmt/goimports for code formatting (mandatory pre-commit)
- Error handling: Always wrap errors with context using fmt.Errorf
- Naming: CamelCase for exported symbols, camelCase for non-exported
- Package organization: One package per directory, package name matches directory
- Use type definitions over string constants for enumerations
- Self-documenting code: Prefer descriptive names over explanatory comments

### Service Layer Design Patterns
- **Command/Query Structure**: Clear separation for user actions in service interfaces
- **Domain Isolation**: Domain interfaces free from infrastructure concerns
- **Dependency Injection**: Constructor pattern for service initialization
- **Context Propagation**: Always pass context.Context for cancellation and timeouts

### Testing Strategy
- **Integration Tests**: testcontainers-go for PostgreSQL database tests
- **Test Utilities**: Shared testing packages (`identitytest/`, `postgrestest/`)
- **Mock Strategy**: Interface-based mocking for external dependencies
- **Test Database**: Isolated test containers per test suite

## Security and Data Protection

### Encryption and Key Management
- **AES-256-GCM**: Credential storage encryption with key versioning
- **Key Derivation**: Environment-based encryption key derivation
- **Secret Management**: No plaintext secrets in logs or database storage
- **Key Rotation**: Support for encryption key versioning and rotation

### Authentication and Authorization
- **Signature Validation**: HMAC-SHA256 and SHA256 for webhook endpoints
- **Session Management**: Secure session handling through Identity Service
- **Organization Context**: Multi-tenant organization isolation

## Integration Architecture Patterns

### Connector System Design
- **Connector Ownership**: Each connector manages its communication method (Socket vs HTTP)
- **Event Isolation**: Connector-specific events, no global event definitions
- **Factory Pattern**: Connector configuration with mapstructure tags
- **Subscribe Pattern**: Event subscription following Clerk authentication pattern

### Code Quality and Type Safety
- **JSON Tags**: Only on API boundary structs (request/response types)
- **Internal Domain**: Clean structs without serialization concerns
- **Modern Go**: Use `any` instead of `interface{}` for type safety
- **Event Extensibility**: Include raw event data (`RawEvent`, `RawPayload`)

### Code Patterns
```go
// ✅ Backend Service Method Pattern
func (s *Service) ProcessSlackMessage(ctx context.Context, cmd ProcessMessageCommand) error {
    // Business logic with proper error wrapping
    if err := s.validateMessage(cmd.Message); err != nil {
        return fmt.Errorf("message validation failed: %w", err)
    }
    return s.forwardToAgent(ctx, cmd)
}

// ✅ Clean Domain Entity
type Integration struct {
    ID             string
    OrganizationID string
    ConnectorType  ConnectorType
    Status         IntegrationStatus
    CreatedAt      time.Time
}

// ✅ API Boundary with JSON Tags
type CreateIntegrationRequest struct {
    OrganizationID string `json:"organization_id"`
    ConnectorType  string `json:"connector_type"`
    Config         any    `json:"config"`
}
```

## Few rules
- Do not add json tags to every struct, we only needs json tags when we need to serialize/deserialize the struct to/from json. ex. api handlers, external json api processing etc. 
- Do not use `interface{}` in the backend, use `any` instead