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

package websocket

import (
	"context"

	ws "github.com/vasu1124/introspect/pkg/operator/websocket"

	introspectv1alpha1 "github.com/vasu1124/introspect/pkg/apis/introspect/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileUselessMachine reconciles a Useless object
type ReconcileUselessMachine struct {
	client.Client
	notifier *ws.Notfier
}

// NewReconcileUselessMachine ...
func NewReconcileUselessMachine(n *ws.Notfier) *ReconcileUselessMachine {
	return &ReconcileUselessMachine{notifier: n}
}

// Reconcile reads that state of the cluster for a Useless object and makes changes based on the state read
// and what is in the Useless.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselesses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselesses/status,verbs=update;patch
func (r *ReconcileUselessMachine) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ul := &introspectv1alpha1.UselessMachineList{}
	if err := r.Client.List(context.TODO(), ul, &client.ListOptions{}); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.notifier.BroadcastUpdates(ul); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// InjectClient for automatic injection of cached client
func (r *ReconcileUselessMachine) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}
