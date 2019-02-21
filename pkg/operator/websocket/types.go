package websocket

import (
	introspect_v1alpha1 "github.com/vasu1124/introspect/pkg/apis/introspect/v1alpha1"
)

type Message struct {
	Name      string                                  `json:"name,omitempty"`
	Namespace string                                  `json:"namespace,omitempty"`
	State     introspect_v1alpha1.UselessMachineState `json:"state,omitempty"`
}
