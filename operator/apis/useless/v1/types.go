package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//CRD Useless

//UselessResourcePlural .
const UselessResourcePlural = "useless"

// Useless
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=useless
type Useless struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   UselessSpec   `json:"spec"`
	Status UselessStatus `json:"status,omitempty"`
}

//UselessSpec .
type UselessSpec struct {
	DesiredState bool `json:"desiredstate"`
}

//UselessStatus .
type UselessStatus struct {
	State     bool         `json:"state"`
	StateCode UselessState `json:"statecode,omitempty"`
	Message   string       `json:"message,omitempty"`
}

//UselessState .
type UselessState string

//UselessStateXYZ .
const (
	UselessStateCreated UselessState = "Created"
	UselessStateOK      UselessState = "OK"
	UselessStateNOK     UselessState = "NotOK"
)

// UselessList .
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=useless
type UselessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Useless `json:"items"`
}
