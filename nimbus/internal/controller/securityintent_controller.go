/*
Copyright 2024.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	intentv1alpha1 "github.com/anurag-rajawat/tutorials/nimbus/api/v1alpha1"
)

// SecurityIntentReconciler reconciles a SecurityIntent object
type SecurityIntentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=intent.security.nimbus.com,resources=securityintents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=intent.security.nimbus.com,resources=securityintents/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SecurityIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var securityIntent intentv1alpha1.SecurityIntent
	err := r.Get(ctx, req.NamespacedName, &securityIntent)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "failed to fetch SecurityIntent", "securityIntent", req.Name)
			return ctrl.Result{}, err
		}
		logger.Info("SecurityIntent not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	logger.Info("reconciling SecurityIntent", "securityIntent", req.Name)

	if err = r.updateStatus(ctx, &securityIntent); err != nil {
		logger.Error(err, "failed to update SecurityIntent status", "securityIntent.name", req.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecurityIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&intentv1alpha1.SecurityIntent{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *SecurityIntentReconciler) updateStatus(ctx context.Context, existingSecurityIntent *intentv1alpha1.SecurityIntent) error {
	existingSecurityIntent.Status = intentv1alpha1.SecurityIntentStatus{
		Action: existingSecurityIntent.Spec.Intent.Action,
		Status: StatusCreated,
	}
	return r.Status().Update(ctx, existingSecurityIntent)
}
