package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/infragpt/services/mcp/kubernetes/pkg/api"
	"github.com/infragpt/services/mcp/kubernetes/pkg/controller"
	"github.com/infragpt/services/mcp/kubernetes/pkg/k8s"
)

func main() {
	var (
		kubeconfig  string
		masterURL   string
		metricsAddr string
		enableLeaderElection bool
		apiAddr     string
	)
	
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	flag.StringVar(&masterURL, "master", "", "URL to Kubernetes API server")
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "Address for serving metrics")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election")
	flag.StringVar(&apiAddr, "api-addr", ":8081", "Address for serving API")
	flag.Parse()
	
	// Set up logging
	logger := log.New(os.Stdout, "MCP-SERVER: ", log.LstdFlags)
	
	// Initialize Kubernetes client
	clientConfig, err := k8s.NewClientConfig(kubeconfig, masterURL)
	if err != nil {
		logger.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	
	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Set up controllers
	controllerManager, err := controller.NewManager(ctx, clientConfig, controller.Options{
		MetricsAddr:          metricsAddr,
		EnableLeaderElection: enableLeaderElection,
		LeaderElectionID:     "mcp-controller-lock",
	})
	if err != nil {
		logger.Fatalf("Unable to set up controller manager: %s", err.Error())
	}
	
	// Start the API server
	apiServer := api.NewServer(apiAddr, clientConfig)
	go func() {
		if err := apiServer.Start(ctx); err != nil {
			logger.Fatalf("Error starting API server: %s", err.Error())
		}
	}()
	
	// Handle shutdown gracefully
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	
	// Start the controller manager
	go func() {
		if err := controllerManager.Start(); err != nil {
			logger.Fatalf("Error starting controller manager: %s", err.Error())
		}
	}()
	
	logger.Printf("MCP server started, waiting for signal to exit")
	
	// Wait for shutdown signal
	<-signalCh
	logger.Println("Received shutdown signal, shutting down gracefully...")
	
	// Give controllers time to shutdown gracefully
	cancelCtx, cancelFunc := context.WithTimeout(ctx, 5*time.Second)
	defer cancelFunc()
	
	if err := apiServer.Shutdown(cancelCtx); err != nil {
		logger.Printf("Error during API server shutdown: %s", err.Error())
	}
	
	cancel() // Signal controllers to shutdown
	
	logger.Println("Shutdown complete")
}