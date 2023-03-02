package websocket

import (
	"context"
	"encoding/json"

	"github.com/olahol/melody"
	"github.com/vasu1124/introspect/pkg/logger"
	uselessmachinev1alpha1 "github.com/vasu1124/introspect/pkg/operator/useless/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var uiMessage = "State updated by UI"

// Notifier struct
type Notifier struct {
	websocket *melody.Melody
	client    client.Client
}

// NewNotifier .
func NewNotifier(m *melody.Melody, c client.Client) *Notifier {
	m.HandleConnect(func(s *melody.Session) {
		ul := &uselessmachinev1alpha1.UselessMachineList{}
		if err := c.List(context.TODO(), ul, &client.ListOptions{}); err != nil {
			logger.Log.Error(err, "[operatorws] can't list ueslessmachines")
			return
		}
		bb, err := json.Marshal(ul)
		if err != nil {
			logger.Log.Error(err, "[operatorws] can't marshal useless uselessmachinelist")
			return
		}
		if err := s.Write(bb); err != nil {
			logger.Log.Error(err, "[operatorws] can't send updates to websocket")
			return
		}
	})
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		message := &Message{}
		if err := json.Unmarshal(msg, message); err != nil {
			logger.Log.Error(err, "[operatorws] can't get unmarshal message")
			return
		}
		ctx := context.TODO()
		useless := &uselessmachinev1alpha1.UselessMachine{}
		if err := c.Get(ctx, client.ObjectKey{Name: message.Name, Namespace: message.Namespace}, useless); err != nil {
			logger.Log.Error(err, "[operatorws] can't get uselessmachine")
			return
		}
		if useless.Status.ActualState == nil ||
			useless.Status.Message == nil ||
			*useless.Status.ActualState != message.State ||
			*useless.Status.Message != uiMessage {

			useless.Status.ActualState = &message.State
			useless.Status.Message = &uiMessage
			if err := c.Status().Update(ctx, useless); err != nil {
				logger.Log.Error(err, "[operatorws] can't update uselessmachine")
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
