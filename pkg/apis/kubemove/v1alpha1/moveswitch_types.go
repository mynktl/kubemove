package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MoveSwitchSpec defines the desired state of MoveSwitch
// +k8s:openapi-gen=true
type MoveSwitchSpec struct {
	// MoveEngine is name of the engine which needs to
	// be switched(/activated) to remote cluster
	MoveEngine string `json:"moveEngine"`

	// Active defines if this resource will be processed or not
	Active bool `json:"active"`
}

// MoveSwitchStatus defines the observed state of MoveSwitch
// +k8s:openapi-gen=true
type MoveSwitchStatus struct {
	// Stage defines progress stage of MoveSwitch
	// +kubebuilder:validation:Enum=Init;InProgress;Errored;Completed
	Stage string `json:"stage"`

	// Status defines final stage/status of MoveSwitch
	// +kubebuilder:validation:Enum=Init;Done;Failed
	Status string `json:"status"`

	// Reason is an error message
	Reason string `json:"reason"`

	// Volumes is list of volumes with status
	Volumes []*VolumeStatus `json:"volumes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MoveSwitch is the Schema for the moveswitches API
// +k8s:openapi-gen=true
type MoveSwitch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MoveSwitchSpec   `json:"spec,omitempty"`
	Status MoveSwitchStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MoveSwitchList contains a list of MoveSwitch
type MoveSwitchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MoveSwitch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MoveSwitch{}, &MoveSwitchList{})
}
