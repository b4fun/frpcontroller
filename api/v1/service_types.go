package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServicePortProtocol defines the protocol to use.
// +kubebuilder:validation:Enum=TCP;UDP
type ServicePortProtocol string

func (s ServicePortProtocol) ToCorev1Protocol() corev1.Protocol {
	return corev1.Protocol(s)
}

const (
	ServicePortTCP ServicePortProtocol = "TCP"
	ServicePortUDP ServicePortProtocol = "UDP"
)

type ServicePort struct {

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
	// NOTE: pattern copied from `k8s/validation/dns1123LabelFmt#de75bf944306`

	// The name of this port to use in frp side.
	Name string `json:"name"`

	// The protocol to use.
	Protocol ServicePortProtocol `json:"protocol"`

	// The local port to expose (service.ports.TargetPort).
	LocalPort int32 `json:"localPort"`

	// The remote port to use (service.ports.Port).
	RemotePort int32 `json:"remotePort"`
}

func (p ServicePort) ToCorev1ServicePort() corev1.ServicePort {
	return corev1.ServicePort{
		Protocol:   p.Protocol.ToCorev1Protocol(),
		Name:       p.Name,
		Port:       p.RemotePort,
		TargetPort: intstr.FromInt(int(p.LocalPort)),
	}
}

// ServiceSpec defines the desired state of Service
type ServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinLength=1

	// Name of the remote endpoint to use.
	Endpoint string `json:"endpoint"`

	// List of ports that are exposed to the frp server.
	// +patchMergeKey=port
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=remotePort
	// +listMapKey=protocol
	Ports []ServicePort `json:"ports"`

	// The selector for picking up pods to the service.
	Selector map[string]string `json:"selector"`

	// Extra labels for the generated service.
	ServiceLabels map[string]string `json:"serviceLabels,omitempty"`
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
