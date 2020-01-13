package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd/api"
)

// MovePairSpec defines the desired state of MovePair
// +k8s:openapi-gen=true
type MovePairSpec struct {
	// Config is remote cluster config
	Config api.Config `json:"config"`
}

// MovePairStatus defines the observed state of MovePair
// +k8s:openapi-gen=true
type MovePairStatus struct {
	// State defines remote cluster connectivity
	// +kubebuilder:validation:Enum=Errored;Connected
	State string `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MovePair is the Schema for the movepairs API
// +k8s:openapi-gen=true
type MovePair struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MovePairSpec   `json:"spec,omitempty"`
	Status MovePairStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MovePairList contains a list of MovePair
type MovePairList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MovePair `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MovePair{}, &MovePairList{})
}
