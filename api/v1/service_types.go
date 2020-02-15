package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServicePortProtocol defines the protocol to use.
// +kubebuilder:validation:Enum=TCP;UDP
type ServicePortProtocol string

const (
	ServicePortTCP ServicePortProtocol = "TCP"
	ServicePortUDP ServicePortProtocol = "UDP"
)

type ServicePort struct {

	// +kubebuilder:validation:MinLength=1

	// The name of this port to use in frp side.
	Name string `json:"name"`

	// The protocol to use.
	Protocol ServicePortProtocol `json:"protocol"`

	// The local port to expose (service.ports.TargetPort).
	LocalPort int32 `json:"localPort"`

	// The remote port to use (service.ports.Port).
	RemotePort int32 `json:"remotePort"`
}

// ServiceSpec defines the desired state of Service
type ServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinLength=1

	// Name of the remote endpoint to use.
	Endpoint string `json:"endpoint"`

	// The list of ports that are exposed to the frp server.
	// +patchMergeKey=port
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=remotePort
	// +listMapKey=protocol
	Ports []ServicePort `json:"ports"`

	// The selector for picking up pods to the service.
	Selector map[string]string `json:"selector"`
}

type ServiceState string

const (
	ServiceStateActive   ServiceState = "active"
	ServiceStateInactive ServiceState = "inactive"
)

// ServiceStatus defines the observed state of Service
type ServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// State tells the service state.
	// +optional
	State ServiceState `json:"state,omitempty"`
}

// +kubebuilder:object:root=true

// Service is the Schema for the services API
// +kubebuilder:subresource:status
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceSpec   `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
