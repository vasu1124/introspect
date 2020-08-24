package websocket

import (
	uselessmachinev1alpha1 "github.com/vasu1124/introspect/pkg/operator/useless/api/v1alpha1"
)

//Message struct
type Message struct {
	Name      string                                     `json:"name,omitempty"`
	Namespace string                                     `json:"namespace,omitempty"`
	State     uselessmachinev1alpha1.UselessMachineState `json:"state,omitempty"`
}
