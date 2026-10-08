package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=asl;plural=apiserverlogs

type APIServerLog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APIServerLogSpec   `json:"spec,omitempty"`
	Status APIServerLogStatus `json:"status,omitempty"`
}

type APIServerLogSpec struct {
	// +kubebuilder:validation:Optional
	// Namespace where kube-apiserver runs (default: kube-system)
	Namespace string `json:"namespace,omitempty"`

	// +kubebuilder:validation:Optional
	// Container name to read logs from (default: kube-apiserver)
	Container string `json:"container,omitempty"`

	// +kubebuilder:validation:Optional
	// Number of log lines to fetch (default: 200)
	TailLines *int64 `json:"tailLines,omitempty"`
}

type APIServerLogStatus struct {
	// LastFetched is the timestamp of the last successful log fetch
	LastFetched metav1.Time `json:"lastFetched,omitempty"`

	// LastError holds the error message if the last fetch failed
	LastError string `json:"lastError,omitempty"`
}

// +kubebuilder:object:root=true
type APIServerLogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []APIServerLog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&APIServerLog{}, &APIServerLogList{})
}
