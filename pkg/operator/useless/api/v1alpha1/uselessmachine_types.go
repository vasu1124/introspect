// SPDX-FileCopyrightText: 2018 vasu1124
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:generate controller-gen object paths=$GOFILE

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UselessMachineSpec defines the desired state of a UselessMachine
type UselessMachineSpec struct {
	// Desired state of cluster, can only be On or Off
	DesiredState UselessMachineState `json:"desiredState"`
}

// UselessMachineStatus defines the observed state of a UselessMachine
type UselessMachineStatus struct {
	// +optional
	ActualState *UselessMachineState `json:"actualState,omitempty"`
	// +optional
	Message *string `json:"message,omitempty"`
}

// UselessMachine is the Schema for the useless API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Desired",type="string",JSONPath=".spec.desiredState",description="Desired state"
// +kubebuilder:printcolumn:name="Actual",type="string",JSONPath=".status.actualState",description="Actual state"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message",description="Controller message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type UselessMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UselessMachineSpec   `json:"spec,omitempty"`
	Status UselessMachineStatus `json:"status,omitempty"`
}

// UselessMachineList contains a list of UselessMachine
// +kubebuilder:object:root=true
type UselessMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UselessMachine `json:"items"`
}

// UselessMachineState describes the state
// +kubebuilder:validation:Enum=On;Off
type UselessMachineState string

// UselessMachineState Enum
const (
	UselessMachineStateOn  UselessMachineState = "On"
	UselessMachineStateOff UselessMachineState = "Off"
)

func init() {
	SchemeBuilder.Register(&UselessMachine{}, &UselessMachineList{})
}
