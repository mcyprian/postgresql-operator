package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgreSQLList contains a list of PostgreSQL
type PostgreSQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQL `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgreSQL is the Schema for the postgresqls API
// +k8s:openapi-gen=true
type PostgreSQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLSpec   `json:"spec,omitempty"`
	Status PostgreSQLStatus `json:"status,omitempty"`
}

type ManagementState string

const (
	// Managed means that the operator is actively managing its resources and trying to keep the component active.
	ManagementStateManaged = "managed"
	// Unmanaged means that the operator will not take any action related to the component
	ManagementStateUnmanaged = "unmanaged"
)

// PostgreSQLSpec defines the desired state of PostgreSQL
// +k8s:openapi-gen=true
type PostgreSQLSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	ManagementState ManagementState  `json:"managementState"`
	Nodes           []PostgreSQLNode `json:"nodes"`
}

// NodeRole defines role of the individual node
type NodeRole string

const (
	PrimaryRole = "primary"
	StandbyRole = "standby"
)

// PostgreSQLNode defines individual node in PostgreSQL cluster
// +k8s:openapi-gen=true
type PostgreSQLNode struct {
	Role      NodeRole                    `json:"role"`
	Image     string                      `json:"image,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources"`
	Storage   PostgreSQLStorageSpec       `json:"storage"`
	GenUUID   *string                     `json:"genUUID,omitempty"`
}

type PostgreSQLStorageSpec struct {
	StorageClassName *string            `json:"storageClassName,omitempty"`
	Size             *resource.Quantity `json:"size,omitempty"`
}

// PostgreSQLStatus defines the observed state of PostgreSQL
// +k8s:openapi-gen=true
type PostgreSQLStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Nodes []string `json:"nodes"`
}

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
