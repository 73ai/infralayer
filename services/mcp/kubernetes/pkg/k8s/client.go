package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientConfig provides access to the Kubernetes API
type ClientConfig struct {
	RestConfig *rest.Config
	Clientset  *kubernetes.Clientset
}

// NewClientConfig creates a new Kubernetes client configuration
func NewClientConfig(kubeconfigPath, masterURL string) (*ClientConfig, error) {
	var config *rest.Config
	var err error
	
	// Try to use in-cluster config if no kubeconfig is provided
	if kubeconfigPath == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		// Use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}
	
	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	
	return &ClientConfig{
		RestConfig: config,
		Clientset:  clientset,
	}, nil
}