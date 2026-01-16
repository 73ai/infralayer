package controller

import (
	"context"
	"fmt"

	"github.com/infragpt/services/mcp/kubernetes/pkg/k8s"
)

// Options contains options for the controller manager
type Options struct {
	MetricsAddr          string
	EnableLeaderElection bool
	LeaderElectionID     string
}

// Manager manages a set of controllers
type Manager struct {
	clientConfig  *k8s.ClientConfig
	options       Options
	controllers   []Controller
	ctx           context.Context
}

// Controller defines the interface for a controller
type Controller interface {
	Start(ctx context.Context) error
	Name() string
}

// NewManager creates a new controller manager
func NewManager(ctx context.Context, clientConfig *k8s.ClientConfig, options Options) (*Manager, error) {
	manager := &Manager{
		clientConfig: clientConfig,
		options:      options,
		ctx:          ctx,
	}
	
	// Register controllers
	serviceController, err := NewServiceController(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create service controller: %w", err)
	}
	
	manager.RegisterController(serviceController)
	
	return manager, nil
}

// RegisterController registers a new controller
func (m *Manager) RegisterController(controller Controller) {
	m.controllers = append(m.controllers, controller)
}

// Start starts all controllers
func (m *Manager) Start() error {
	for _, controller := range m.controllers {
		go func(c Controller) {
			if err := c.Start(m.ctx); err != nil {
				fmt.Printf("Error starting controller %s: %v\n", c.Name(), err)
			}
		}(controller)
	}
	
	<-m.ctx.Done()
	return nil
}