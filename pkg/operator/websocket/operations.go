package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	uselessmachinev1alpha1 "github.com/vasu1124/introspect/pkg/operator/useless/api/v1alpha1"
	melody "gopkg.in/olahol/melody.v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var uiMessage = "State updated by UI"

//Notifier struct
type Notifier struct {
	websocket *melody.Melody
	client    client.Client
}

// NewNotifier .
func NewNotifier(m *melody.Melody, c client.Client) *Notifier {
	m.HandleConnect(func(s *melody.Session) {
		ul := &uselessmachinev1alpha1.UselessMachineList{}
		if err := c.List(context.TODO(), ul, &client.ListOptions{}); err != nil {
			fmt.Printf("can't list ueslessmachines: %v", err)
			return
		}
		bb, err := json.Marshal(ul)
		if err != nil {
			fmt.Printf("can't marshal useless uselessmachinelist: %v", err)
			return
		}
		if err := s.Write(bb); err != nil {
			fmt.Printf("can't send updates to websocket: %v", err)
			return
		}
	})
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		message := &Message{}
		if err := json.Unmarshal(msg, message); err != nil {
			fmt.Printf("can't get unmarshal message: %v", err)
			return
		}
		ctx := context.TODO()
		useless := &uselessmachinev1alpha1.UselessMachine{}
		if err := c.Get(ctx, client.ObjectKey{Name: message.Name, Namespace: message.Namespace}, useless); err != nil {
			fmt.Printf("can't get uselessmachine: %v", err)
			return
		}
		if useless.Status.ActualState == nil ||
			useless.Status.Message == nil ||
			*useless.Status.ActualState != message.State ||
			*useless.Status.Message != uiMessage {

			useless.Status.ActualState = &message.State
			useless.Status.Message = &uiMessage
			if err := c.Status().Update(ctx, useless); err != nil {
				fmt.Printf("can't update uselessmachine: %v", err)
				return
			}
		}

	})
	return &Notifier{m, c}
}

// BroadcastUpdates to all
func (n *Notifier) BroadcastUpdates(ul *uselessmachinev1alpha1.UselessMachineList) error {
	b, err := json.Marshal(ul)
	if err != nil {
		return err
	}
	return n.websocket.Broadcast(b)
}
