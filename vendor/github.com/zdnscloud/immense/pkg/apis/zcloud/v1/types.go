package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSpec defines the desired state of Cluster
// +k8s:openapi-gen=true
type ClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	StorageType string `json:"storageType"`
	//Hosts       []HostSpec `json:"hosts"`
	Hosts []string `json:"hosts"`
}

type HostInfo struct {
	NodeName     string `json:"nodeName"`
	BlockDevices []Dev  `json:"blockDevices"`
}

type Dev struct {
	Name string `json:"name"`
	Size string `json:"szie"`
}

// ClusterStatus defines the observed state of Cluster
// +k8s:openapi-gen=true
type ClusterStatus struct {
	Phase    string     `json:"phase,omitempty"`
	Message  string     `json:"message,omitempty"`
	Config   []HostInfo `json:"config"`
	Capacity `json:"capacity,omitempty"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

type Capacity struct {
	Total     Size       `json:"total"`
	Instances []Instance `json:"instances"`
}

type Instance struct {
	Host string `json:"host"`
	Dev  string `json:"dev"`
	Stat bool   `json:"stat"`
	Info Size   `json:"info"`
}

type Size struct {
	Total string `json:"total"`
	Used  string `json:"used"`
	Free  string `json:"free"`
}
