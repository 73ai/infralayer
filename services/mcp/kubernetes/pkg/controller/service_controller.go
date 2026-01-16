package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	
	"github.com/infragpt/services/mcp/kubernetes/pkg/k8s"
	"github.com/infragpt/services/mcp/kubernetes/pkg/model"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceController manages MCP services
type ServiceController struct {
	clientConfig *k8s.ClientConfig
	services     map[string]model.MCPService
	mutex        sync.RWMutex
	logger       *log.Logger
}

// NewServiceController creates a new service controller
func NewServiceController(clientConfig *k8s.ClientConfig) (*ServiceController, error) {
	logger := log.New(os.Stdout, "SERVICE-CONTROLLER: ", log.LstdFlags)
	
	return &ServiceController{
		clientConfig: clientConfig,
		services:     make(map[string]model.MCPService),
		logger:       logger,
	}, nil
}

// Name returns the controller name
func (c *ServiceController) Name() string {
	return "service-controller"
}

// Start starts the service controller
func (c *ServiceController) Start(ctx context.Context) error {
	c.logger.Println("Starting service controller")
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	// Initial reconciliation
	if err := c.reconcile(ctx); err != nil {
		c.logger.Printf("Initial reconciliation failed: %v", err)
	}
	
	for {
		select {
		case <-ctx.Done():
			c.logger.Println("Shutting down service controller")
			return nil
		case <-ticker.C:
			if err := c.reconcile(ctx); err != nil {
				c.logger.Printf("Reconciliation failed: %v", err)
			}
		}
	}
}

// reconcile reconciles the desired state with the actual state
func (c *ServiceController) reconcile(ctx context.Context) error {
	c.logger.Println("Reconciling services")
	
	// In a real implementation, this would:
	// 1. List MCP services from a custom resource definition (CRD)
	// 2. For each service, ensure the corresponding Kubernetes resources exist
	// 3. Update status based on the actual state
	
	// Mock reconciliation for demonstration
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	for key, service := range c.services {
		// Simulate reconciliation by updating status
		
		// Check if service needs to be deployed
		if service.Status.Phase == model.ServicePending {
			c.logger.Printf("Deploying service %s", service.ObjectMeta.Name)
			
			service.Status.Phase = model.ServiceDeploying
			service.Status.Conditions = append(service.Status.Conditions, model.ServiceCondition{
				Type:               model.ServiceProgressing,
				Status:             model.ConditionTrue,
				LastUpdateTime:     metav1.Now(),
				LastTransitionTime: metav1.Now(),
				Reason:             "Deploying",
				Message:            fmt.Sprintf("Deploying service %s", service.ObjectMeta.Name),
			})
			
			c.services[key] = service
			
			// Simulate deployment time
			time.Sleep(2 * time.Second)
			
			// Update status to running after "deployment"
			service.Status.Phase = model.ServiceRunning
			service.Status.AvailableReplicas = service.Spec.Replicas
			service.Status.Conditions = append(service.Status.Conditions, model.ServiceCondition{
				Type:               model.ServiceAvailable,
				Status:             model.ConditionTrue,
				LastUpdateTime:     metav1.Now(),
				LastTransitionTime: metav1.Now(),
				Reason:             "ServiceDeployed",
				Message:            fmt.Sprintf("Service %s successfully deployed", service.ObjectMeta.Name),
			})
			
			c.services[key] = service
			c.logger.Printf("Service %s deployed successfully", service.ObjectMeta.Name)
		}
	}
	
	return nil
}

// AddService adds a service to be managed by the controller
func (c *ServiceController) AddService(service model.MCPService) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	key := fmt.Sprintf("%s/%s", service.ObjectMeta.Namespace, service.ObjectMeta.Name)
	c.services[key] = service
	
	c.logger.Printf("Added service %s to controller", key)
}

// GetService gets a service by name and namespace
func (c *ServiceController) GetService(namespace, name string) (model.MCPService, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	key := fmt.Sprintf("%s/%s", namespace, name)
	service, exists := c.services[key]
	return service, exists
}

// ListServices lists all services
func (c *ServiceController) ListServices() []model.MCPService {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	services := make([]model.MCPService, 0, len(c.services))
	for _, service := range c.services {
		services = append(services, service)
	}
	
	return services
}

// DeleteService deletes a service
func (c *ServiceController) DeleteService(namespace, name string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	key := fmt.Sprintf("%s/%s", namespace, name)
	_, exists := c.services[key]
	if exists {
		delete(c.services, key)
		c.logger.Printf("Deleted service %s", key)
	}
	
	return exists
}