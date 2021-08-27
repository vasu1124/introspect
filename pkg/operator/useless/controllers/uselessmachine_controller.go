/*


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

package controllers

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	controller_runtime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uselessmachinev1alpha1 "github.com/vasu1124/introspect/pkg/operator/useless/api/v1alpha1"

	ws "github.com/vasu1124/introspect/pkg/operator/websocket"
)

// UselessMachineReconciler reconciles a UselessMachine object
type UselessMachineReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Notifier *ws.Notifier
}

// Reconcile reads that state of the cluster for a UselessMachine object and makes changes based on the state read and what is in the Useless.Spec
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselessmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselessmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselessmachines/finalizers,verbs=update
func (r *UselessMachineReconciler) Reconcile(ctx context.Context, req controller_runtime.Request) (controller_runtime.Result, error) {
	//	ctx := context.Background()
	log := r.Log.WithValues("uselessmachine", req.NamespacedName)

	if r.Notifier != nil {
		ul := &uselessmachinev1alpha1.UselessMachineList{}
		if err := r.Client.List(ctx, ul, &client.ListOptions{}); err != nil {
			return controller_runtime.Result{}, err
		}

		if err := r.Notifier.BroadcastUpdates(ul); err != nil {
			return controller_runtime.Result{}, err
		}
	}

	// Fetch the Useless instance
	instance := &uselessmachinev1alpha1.UselessMachine{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.V(4).Info("unable to fetch UselessMachine")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		// Object not found, return. Created objects are automatically garbage collected.
		// For additional cleanup logic use finalizers.
		return controller_runtime.Result{}, client.IgnoreNotFound(err)
	}

	desiredStatus := *r.desiredStatus(instance)
	if !reflect.DeepEqual(instance.Status, desiredStatus) {
		go func() {
			log.V(1).Info("Updating UselessMachine", "Status", instance.Status.ActualState, "DesiredStatus", desiredStatus.ActualState)
			// We pretend that the update take 4sec, otherwise demonstrating is futile
			time.Sleep(4 * time.Second)
			instance.Status = desiredStatus
			if err := r.Status().Update(ctx, instance); err != nil {
				log.V(3).Info("Unable to update UselessMachine status")
			}
		}()
		return controller_runtime.Result{}, nil

	}

	return controller_runtime.Result{}, nil
}

// SetupWithManager .
func (r *UselessMachineReconciler) SetupWithManager(mgr controller_runtime.Manager) error {
	return controller_runtime.NewControllerManagedBy(mgr).
		For(&uselessmachinev1alpha1.UselessMachine{}).
		Complete(r)
}

func (r *UselessMachineReconciler) desiredStatus(u *uselessmachinev1alpha1.UselessMachine) *uselessmachinev1alpha1.UselessMachineStatus {
	message := "State updated by introspect"
	desiredState := u.Spec.DesiredState
	return &uselessmachinev1alpha1.UselessMachineStatus{
		Message:     &message,
		ActualState: &desiredState,
	}
}

// InjectClient for automatic injection of cached client
func (r *UselessMachineReconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}
