package model

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MCPService represents a service managed by the MCP
type MCPService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCPServiceSpec   `json:"spec,omitempty"`
	Status MCPServiceStatus `json:"status,omitempty"`
}

// MCPServiceSpec defines the desired state of an MCP service
type MCPServiceSpec struct {
	// Type defines the service type
	Type string `json:"type"`
	
	// Version defines the service version
	Version string `json:"version"`
	
	// Replicas is the desired number of replicas
	Replicas int32 `json:"replicas"`
	
	// Image is the container image to use
	Image string `json:"image"`
	
	// Resources defines the resource requirements
	Resources ResourceRequirements `json:"resources,omitempty"`
	
	// Config holds service-specific configuration
	Config map[string]string `json:"config,omitempty"`
}

// ResourceRequirements defines resource requirements for the service
type ResourceRequirements struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// MCPServiceStatus defines the observed state of an MCP service
type MCPServiceStatus struct {
	// Phase represents the current lifecycle phase of the service
	Phase ServicePhase `json:"phase"`
	
	// AvailableReplicas is the number of available replicas
	AvailableReplicas int32 `json:"availableReplicas"`
	
	// Conditions represents the latest available observations of the service's state
	Conditions []ServiceCondition `json:"conditions,omitempty"`
}

// ServicePhase is a label for the condition of a service at the current time
type ServicePhase string

const (
	// ServicePending means the service has been accepted by the system, but deployment is pending
	ServicePending ServicePhase = "Pending"
	
	// ServiceDeploying means the deployment is in progress
	ServiceDeploying ServicePhase = "Deploying"
	
	// ServiceRunning means the service is operational
	ServiceRunning ServicePhase = "Running"
	
	// ServiceFailed means the service is not operational
	ServiceFailed ServicePhase = "Failed"
)

// ServiceCondition describes the state of a service at a certain point
type ServiceCondition struct {
	// Type of service condition
	Type ServiceConditionType `json:"type"`
	
	// Status of the condition, one of True, False, Unknown
	Status ConditionStatus `json:"status"`
	
	// Last time the condition was updated
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	
	// Last time the condition transitioned from one status to another
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	
	// The reason for the condition's last transition
	Reason string `json:"reason,omitempty"`
	
	// A human-readable message indicating details about the transition
	Message string `json:"message,omitempty"`
}

// ServiceConditionType is a valid value for ServiceCondition.Type
type ServiceConditionType string

const (
	// ServiceAvailable means the service is available and ready to accept requests
	ServiceAvailable ServiceConditionType = "Available"
	
	// ServiceProgressing means the deployment is progressing
	ServiceProgressing ServiceConditionType = "Progressing"
)

// ConditionStatus is the status of a condition
type ConditionStatus string

const (
	// ConditionTrue means a condition is true
	ConditionTrue ConditionStatus = "True"
	
	// ConditionFalse means a condition is false
	ConditionFalse ConditionStatus = "False"
	
	// ConditionUnknown means the system cannot determine the status of a condition
	ConditionUnknown ConditionStatus = "Unknown"
)