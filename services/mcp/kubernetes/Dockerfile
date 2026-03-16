FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy the go module files first to leverage Docker cache
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mcp-server ./cmd/server

# Use distroless for minimal image
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/mcp-server .
COPY --from=builder /app/config /app/config

# Run as non-root user
USER nonroot:nonroot

EXPOSE 8080 8081

ENTRYPOINT ["/app/mcp-server"]