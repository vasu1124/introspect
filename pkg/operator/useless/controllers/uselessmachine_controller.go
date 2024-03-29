// SPDX-FileCopyrightText: 2018 vasu1124
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uselessmachinev1alpha1 "github.com/vasu1124/introspect/pkg/operator/useless/api/v1alpha1"
	ws "github.com/vasu1124/introspect/pkg/operator/websocket"
)

// UselessMachineReconciler reconciles a UselessMachine object
type UselessMachineReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Notifier *ws.Notifier
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselessmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselessmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=introspect.actvirtual.com,resources=uselessmachines/finalizers,verbs=update
func (r *UselessMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log, _ := logr.FromContext(ctx)

	if r.Notifier != nil {
		ul := &uselessmachinev1alpha1.UselessMachineList{}
		if err := r.Client.List(ctx, ul, &client.ListOptions{}); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Notifier.BroadcastUpdates(ul); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Fetch the Useless instance
	instance := &uselessmachinev1alpha1.UselessMachine{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Info("[controller] unable to fetch UselessMachine")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		// Object not found, return. Created objects are automatically garbage collected.
		// For additional cleanup logic use finalizers.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	desiredStatus := *r.desiredStatus(instance)
	if !reflect.DeepEqual(instance.Status, desiredStatus) {
		go func() {
			log.Info("[controller] Updating UselessMachine", "Status", instance.Status.ActualState, "DesiredStatus", desiredStatus.ActualState)
			// We pretend that the update take 4sec, otherwise demonstrating is futile
			time.Sleep(4 * time.Second)
			instance.Status = desiredStatus
			if err := r.Status().Update(ctx, instance); err != nil {
				log.Info("[controller] Unable to update UselessMachine status")
			}
		}()
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UselessMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
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
