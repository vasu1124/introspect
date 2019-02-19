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

package useless

import (
	"context"
	"log"
	"reflect"
	"time"

	introspectv1alpha1 "github.com/vasu1124/introspect/pkg/apis/introspect/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileUselessMachine reconciles a UselessMachine object
type ReconcileUselessMachine struct {
	client.Client
}

// Reconcile reads that state of the cluster for a UselessMachine object and makes changes based on the state read
// and what is in the Useless.Spec
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselessmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselessmachines/status,verbs=update;patch
func (r *ReconcileUselessMachine) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Useless instance
	instance := &introspectv1alpha1.UselessMachine{}
	if err := r.Get(context.TODO(), request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	desiredStatus := *r.desiredStatus(instance)
	if !reflect.DeepEqual(instance.Status, desiredStatus) {
		log.Printf("Updating Useless %s/%s\n", instance.Namespace, instance.Name)
		// TODO: externalize this
		time.Sleep(4 * time.Second)
		instance.Status = desiredStatus
		if err := r.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileUselessMachine) desiredStatus(u *introspectv1alpha1.UselessMachine) *introspectv1alpha1.UselessMachineStatus {
	message := "Successfully corrected by introspect controller"
	desiredState := u.Spec.DesiredState
	return &introspectv1alpha1.UselessMachineStatus{
		Message:     &message,
		ActualState: &desiredState,
	}
}

// InjectClient for automatic injection of cached client
func (r *ReconcileUselessMachine) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}
