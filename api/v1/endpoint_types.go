package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EndpointSpec defines the desired state of Endpoint
type EndpointSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinLength=1

	// Addr specifies the remote endpoint address.
	Addr string `json:"addr"`

	// +kubebuilder:validation:Min=0

	// Port specifies the remote port.
	Port int32 `json:"port"`

	// +kubebuilder:validation:MinLength=1

	// Token specifies the token to connect the endpoint.
	// +optional
	Token string `json:"token"`
}

type EndpointState string

const (
	EndpointConnected    EndpointState = "Connected"
	EndpointDisconnected EndpointState = "Disconnected"
)

// EndpointStatus defines the observed state of Endpoint
type EndpointStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// State tells the state of the endpoint.
	// +optional
	State EndpointState `json:"state"`
}

// +kubebuilder:object:root=true

// Endpoint is the Schema for the endpoints API
// +kubebuilder:subresource:status
type Endpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EndpointSpec   `json:"spec,omitempty"`
	Status EndpointStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EndpointList contains a list of Endpoint
type EndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Endpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Endpoint{}, &EndpointList{})
}
