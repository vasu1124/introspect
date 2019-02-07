/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UselessMachineSpec defines the desired state of Useless
type UselessMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Enum=On,Off
	DesiredState UselessMachineState `json:"desiredState"`
}

// UselessMachineStatus defines the observed state of Useless
type UselessMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	// +kubebuilder:validation:Enum=On,Off
	ActualState *UselessMachineState `json:"actualState,omitempty"`
	// +optional
	Message *string `json:"message,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UselessMachine is the Schema for the uselesses API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UselessMachineList contains a list of Useless
type UselessMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UselessMachine `json:"items"`
}

//UselessMachineState .
type UselessMachineState string

//UselessMachineStateXYZ .
const (
	UselessMachineStateOn  UselessMachineState = "On"
	UselessMachineStateOff UselessMachineState = "Off"
)

func init() {
	SchemeBuilder.Register(&UselessMachine{}, &UselessMachineList{})
}
