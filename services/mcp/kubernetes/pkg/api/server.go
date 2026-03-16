package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/infragpt/services/mcp/kubernetes/pkg/k8s"
	"github.com/infragpt/services/mcp/kubernetes/pkg/model"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Server represents the API server
type Server struct {
	addr        string
	clientConfig *k8s.ClientConfig
	server       *http.Server
	logger       *log.Logger
}

// NewServer creates a new API server instance
func NewServer(addr string, clientConfig *k8s.ClientConfig) *Server {
	logger := log.New(os.Stdout, "API-SERVER: ", log.LstdFlags)
	
	return &Server{
		addr:        addr,
		clientConfig: clientConfig,
		logger:       logger,
	}
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// Register API endpoints
	mux.HandleFunc("/api/v1/health", s.healthHandler)
	mux.HandleFunc("/api/v1/services", s.servicesHandler)
	
	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	
	s.logger.Printf("Starting API server on %s", s.addr)
	
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("error starting server: %w", err)
	}
	
	return nil
}

// Shutdown gracefully shuts down the API server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Println("Shutting down API server...")
	return s.server.Shutdown(ctx)
}

// healthHandler handles health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	response := map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// servicesHandler handles service-related requests
func (s *Server) servicesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listServices(w, r)
	case http.MethodPost:
		s.createService(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listServices returns a list of all services
func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would fetch services from Kubernetes
	// For now, return a sample service
	services := []model.MCPService{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample-service",
				Namespace: "default",
			},
			Spec: model.MCPServiceSpec{
				Type:     "web",
				Version:  "1.0.0",
				Replicas: 2,
				Image:    "nginx:latest",
			},
			Status: model.MCPServiceStatus{
				Phase:            model.ServiceRunning,
				AvailableReplicas: 2,
			},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

// createService creates a new service
func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	var service model.MCPService
	
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate service
	if service.Spec.Type == "" || service.Spec.Image == "" {
		http.Error(w, "Service type and image are required", http.StatusBadRequest)
		return
	}
	
	// Set default values if not provided
	if service.Spec.Replicas <= 0 {
		service.Spec.Replicas = 1
	}
	
	// In a real implementation, this would create the service in Kubernetes
	// Mock a successful creation
	service.Status = model.MCPServiceStatus{
		Phase:            model.ServicePending,
		AvailableReplicas: 0,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(service)
}