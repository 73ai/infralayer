# InfraLayer Platform

InfraLayer is a multi-service AI SRE platform that converts natural language into infrastructure commands and automates DevOps workflows through intelligent integrations.

## Architecture

The platform follows a services-based architecture with shared modules and a centralized CLI client.

### Services (`/services/`)

#### Backend Service
Backend API server providing core platform functionality and integrations with external services.

#### Web Application  
Frontend interface for skills creation, management, and user interactions.

#### Agent Service
AI-powered agent system for automated DevOps workflows and intelligent task execution.

#### Website
Marketing and documentation website for the platform.

#### MCP (Model Context Protocol)
Protocol implementation for AI model context management and communication.

### CLI Client (`/cli/`)
Interactive command-line interface for natural language to infrastructure command conversion.

### Shared Modules

#### LLM Module (`/llm/`)
Centralized AI functionality shared across all services including client libraries, authentication, and prompt management.

## Navigation

Each service maintains its own documentation and development guidelines in service-specific directories. Refer to individual README files and CLAUDE.md files within each service for detailed implementation information and development instructions.