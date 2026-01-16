# Kubernetes MCP Server

A Management Control Plane (MCP) server for Kubernetes that provides APIs for managing services.

## Overview

The MCP server is a Go-based application that interacts with Kubernetes to provide a higher-level API for managing services. It serves as an abstraction layer between users and Kubernetes, making it easier to deploy and manage applications.

## Features

- RESTful API for service management
- Authentication with JWT and API keys
- Kubernetes controller for service reconciliation
- Metrics and health monitoring
- Deployment as a Kubernetes application

## Project Structure

```
.
├── cmd/                  # Command-line applications
│   └── server/           # MCP server entry point
├── pkg/                  # Library packages
│   ├── api/              # API server and handlers
│   ├── auth/             # Authentication and authorization
│   ├── controller/       # Kubernetes controllers
│   ├── k8s/              # Kubernetes client utilities
│   └── model/            # Data models
├── config/               # Configuration files
├── deploy/               # Deployment manifests
│   └── kubernetes/       # Kubernetes deployment manifests
└── Dockerfile            # Container image definition
```

## Getting Started

### Prerequisites

- Go 1.22 or higher
- Access to a Kubernetes cluster
- kubectl configured to access your cluster

### Building

```bash
# Build the server
go build -o mcp-server ./cmd/server

# Run the server
./mcp-server
```

### Running with Docker

```bash
# Build the Docker image
docker build -t infragpt/mcp-server:latest .

# Run the container
docker run -p 8081:8081 -p 8080:8080 infragpt/mcp-server:latest
```

### Deploying to Kubernetes

```bash
# Create namespace
kubectl apply -f deploy/kubernetes/namespace.yaml

# Create RBAC resources
kubectl apply -f deploy/kubernetes/rbac.yaml

# Create ConfigMap
kubectl apply -f deploy/kubernetes/configmap.yaml

# Deploy the server
kubectl apply -f deploy/kubernetes/deployment.yaml
kubectl apply -f deploy/kubernetes/service.yaml
```

## API Endpoints

- `GET /api/v1/health` - Health check
- `GET /api/v1/services` - List services
- `POST /api/v1/services` - Create a service

## Configuration

Configuration is loaded from `config/config.yaml`. See the example configuration for details.

## Development

### Running Tests

```bash
go test ./...
```

### Code Style

This project follows the standard Go style guidelines and uses `gofmt` for formatting.

## License

This project is licensed under the MIT License - see the LICENSE file for details.