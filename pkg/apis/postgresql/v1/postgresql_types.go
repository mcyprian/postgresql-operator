package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ManagementState string

const (
	// Managed means that the operator is actively managing its resources and trying to keep the component active.
	ManagementStateManaged = "Managed"
	// Unmanaged means that the operator will not take any action related to the component
	ManagementStateUnmanaged = "Unmanaged"
)

// PostgreSQLSpec defines the desired state of PostgreSQL
// +k8s:openapi-gen=true
type PostgreSQLSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	ManagementState ManagementState `json:"managementState"`
	Size            int32           `json:"size"`
}

// PostgreSQLStatus defines the observed state of PostgreSQL
// +k8s:openapi-gen=true
type PostgreSQLStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Nodes []string `json:"nodes"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgreSQLList contains a list of PostgreSQL
type PostgreSQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
