package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataVolume defines volume space for dataSync
type DataVolume struct {
	// Namespace is given volume's claim namespace
	Namespace string `json:"namespace"`

	// Name is given volume name
	Name string `json:"name"`

	// PVC is name of PVC for given volume
	PVC string `json:"pvc"`

	// RemoteName is volume name generated at remote cluster for
	// given volume
	RemoteName string `json:"remoteName"`

	// RemoteNamespace is remote volume's claim namespace
	RemoteNamespace string `json:"remoteNamespace"`

	// Param is key/value pair for given volumes
	Param map[string]string `json:"param,omitempty"`
}

// DataSyncSpec defines the desired state of DataSync
// +k8s:openapi-gen=true
type DataSyncSpec struct {
	// Volume is list having information about volume at source and remote cluster
	Volume []*DataVolume `json:"volume"`

	// Namespace is source cluster's namespace for which migration is running
	Namespace string `json:"namespace"`

	// PluginProvider is plugin name to be used for migration
	// +kubebuilder:validation:MinLength=1
	PluginProvider string `json:"plugin"`

	// MoveEngine is name of engine for which DataSync resource is generated
	// +kubebuilder:validation:MinLength=1
	MoveEngine string `json:"moveEngine"`

	// Active is to decide if current cluster is sender or receiver
	Active bool `json:"backup"`

	// Config is list of parameters for Plugin
	Config map[string]string `json:"config,omitempty"`
}

// DataSyncStatus defines the observed state of DataSync
// +k8s:openapi-gen=true
type DataSyncStatus struct {
	// Stage defines DataSync execution phase
	// +kubebuilder:validation:Enum=Init;Errored;Canceled;InProgress;Done
	Stage string `json:"stage"`

	// Status defines final status of DataSync
	// +kubebuilder:validation:Enum=Init;Done;Failed
	Status string `json:"status"`

	// CompletionTime is time when migration for current DataSync completed
	CompletionTime string `json:"completionTime"`

	// Volumes is list of volume migration status
	Volumes []*VolumeStatus `json:"volume"`

	// Reason is an error message
	Reason string `json:"reason"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataSync is the Schema for the datasyncs API
// +k8s:openapi-gen=true
type DataSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataSyncSpec   `json:"spec,omitempty"`
	Status DataSyncStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataSyncList contains a list of DataSync
type DataSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataSync{}, &DataSyncList{})
}
