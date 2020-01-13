package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MoveEngineSpec defines the specification for Migration
// +k8s:openapi-gen=true
type MoveEngineSpec struct {
	// MovePair is a name of MovePair having remote
	// cluster details
	// +kubebuilder:validation:MinLength=1
	MovePair string `json:"movePair"`

	// Namespace is application namespace
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// RemoteNamespace is namespace for remote cluster.
	// If empty, same Namespace will be used to deploy an application
	RemoteNamespace string `json:"remoteNamespace"`

	// Selectors is to list out application resource for migration
	Selectors *metav1.LabelSelector `json:"selectors"`

	// SyncPeriod is cron base time period defining when to run migration
	// +kubebuilder:validation:MinLength=1
	SyncPeriod string `json:"syncPeriod"`

	// Mode is to decide if current MoveEngine is migration sender or receiver
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Enum=Active;Passive
	Mode string `json:"mode"`

	// PluginProvider is a plugin name which will support the migration for
	// given Engine
	// +kubebuilder:validation:MinLength=1
	PluginProvider string `json:"plugin"`

	// IncludeResources is to check if supporting k8s native resources to
	// be created or not
	IncludeResources bool `json:"includeResources"`
}

// MoveEngineStatus defines the observed state of MoveEngine
// +k8s:openapi-gen=true
type MoveEngineStatus struct {
	// Status is current MoveEngine status
	Status string `json:"Status"`

	// LastStatus is last MoveEngine status
	LastStatus string `json:"LastStatus"`

	// SyncedTime is last synced time
	SyncedTime metav1.Time `json:"SyncedTime"`

	// LastSyncedTime is synced time for second-to-last sync
	LastSyncedTime metav1.Time `json:"LastSyncedTime"`

	// DataSync is name of the DataSync resource generated for
	// current migration cycle
	DataSync string `json:"DataSync"`

	// DataSyncStatus is status of the geerated DataSync resource
	DataSyncStatus string `json:"DataSyncStatus"`

	// Volumes is a list containing migration status for volumes resource
	// This status includes information about volume related resource only, not data
	Volumes []*VolumeStatus `json:"Volumes"`

	// Resources is a list containing migration status for resources other than volumes
	Resources []*ResourceStatus `json:"Resources"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MoveEngine is the Schema for the moveengines API
// +k8s:openapi-gen=true
type MoveEngine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MoveEngineSpec   `json:"spec,omitempty"`
	Status MoveEngineStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MoveEngineList contains a list of MoveEngine
type MoveEngineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MoveEngine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MoveEngine{}, &MoveEngineList{})
}

// VolumeStatus sync status of volumes
type VolumeStatus struct {
	// Namespace is volume claim's namespace
	Namespace string `json:"namespace"`

	// RemoteNamespace is volume claim's namespace for remote cluster
	RemoteNamespace string `json:"remoteNamespace"`

	// PVC name of volume's claim
	PVC string `json:"pvc"`

	// status is volume resource migration status
	Status string `json:"Status"`

	// SyncedTime is time at which last synced for volume resource completed
	SyncedTime metav1.Time `json:"Synced"`

	// LastStatus is second-to-last volume resource sync status
	LastStatus string `json:"lastStatus"`

	// LastSyncedTime is second-to-last synced time for volume resource
	LastSyncedTime metav1.Time `json:"lastSyncedTime"`

	// Reason is an error message
	Reason string `json:"reason"`

	// Volume is volume name
	Volume string `json:"Volume"`

	// RemoteVolume is remote volume name for given volume
	RemoteVolume string `json:"RemoteVolume"`
}

// ResourceStatus sync status of resource
type ResourceStatus struct {
	// Kind defines resource kind
	Kind string `json:"kind"`

	// Name is resource name
	Name string `json:"name"`

	// Phase defines migration phase for given resource
	Phase string `json:"phase"`

	// Status defines migration status for given resource
	Status string `json:"status"`

	// Reason is an error message
	Reason string `json:"reason"`

	// SyncedTime is last sync time for given resource
	SyncedTime metav1.Time `json:"Synced"`
}
