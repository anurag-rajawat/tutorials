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
	"bytes"
	"context"
	"encoding/json"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	intentv1alpha1 "github.com/anurag-rajawat/tutorials/nimbus/api/v1alpha1"
	"github.com/anurag-rajawat/tutorials/nimbus/pkg/builder"
	buildererrors "github.com/anurag-rajawat/tutorials/nimbus/pkg/utils/errors"
)

// SecurityIntentBindingReconciler reconciles a SecurityIntentBinding object
type SecurityIntentBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=intent.security.nimbus.com,resources=securityintentbindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=intent.security.nimbus.com,resources=securityintentbindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=intent.security.nimbus.com,resources=securityintentbindings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SecurityIntentBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var securityIntentBinding intentv1alpha1.SecurityIntentBinding
	err := r.Get(ctx, req.NamespacedName, &securityIntentBinding)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "failed to fetch SecurityIntentBinding", "securityIntentBinding.name", req.Name, "securityIntentBinding.namespace", req.Namespace)
			return ctrl.Result{}, err
		}
		logger.Info("SecurityIntentBinding not found. Ignoring since object must be deleted", "securityIntentBinding.name", req.Name, "securityIntentBinding.namespace", req.Namespace)
		return ctrl.Result{}, nil
	}

	logger.Info("reconciling SecurityIntentBinding", "securityIntentBinding.name", req.Name, "securityIntentBinding.namespace", req.Namespace)

	nimbusPolicy, err := r.createOrUpdateNimbusPolicy(ctx, securityIntentBinding)
	if err != nil {
		return ctrl.Result{}, err
	}

	if nimbusPolicy != nil {
		if err = r.updateNpStatus(ctx, nimbusPolicy); err != nil {
			logger.Error(err, "failed to update NimbusPolicy status", "nimbusPolicy.Name", nimbusPolicy.Name, "nimbusPolicy.Namespace", nimbusPolicy.Namespace)
			return ctrl.Result{}, err
		}

		if err = r.updateSibStatusWithBoundNpAndSisInfo(ctx, &securityIntentBinding, nimbusPolicy); err != nil {
			logger.Error(err, "failed to update SecurityIntentBinding status", "securityIntentBinding.name", req.Name, "securityIntentBinding.namespace", req.Namespace)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecurityIntentBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&intentv1alpha1.SecurityIntentBinding{}).
		Owns(&intentv1alpha1.NimbusPolicy{}).
		Watches(&intentv1alpha1.SecurityIntent{},
			handler.EnqueueRequestsFromMapFunc(r.findBindingsMatchingWithIntent),
		).
		Complete(r)
}

func (r *SecurityIntentBindingReconciler) createOrUpdateNimbusPolicy(ctx context.Context, securityIntentBinding intentv1alpha1.SecurityIntentBinding) (*intentv1alpha1.NimbusPolicy, error) {
	logger := log.FromContext(ctx)

	nimbusPolicyToCreate, err := builder.BuildNimbusPolicy(ctx, r.Client, securityIntentBinding)
	if err != nil {
		if errors.Is(err, buildererrors.ErrSecurityIntentsNotFound) {
			// Since the SecurityIntent(s) referenced in SecurityIntentBinding spec don't
			// exist, so delete NimbusPolicy if it exists.
			if err = r.deleteNimbusPolicyIfExists(ctx, securityIntentBinding.Name, securityIntentBinding.Namespace); err != nil {
				return nil, err
			}

			// When a NimbusPolicy is deleted, it implies the referenced SecurityIntent(s)
			// is(are) no longer exist. Therefore, update the status subresource of the
			// associated SecurityIntentBinding to reflect the latest details.
			if err = r.removeNpAndSisDetailsFromSibStatus(ctx, securityIntentBinding.Name, securityIntentBinding.Namespace); err != nil {
				return nil, err
			}

			return nil, nil
		}
		return nil, err
	}

	var nimbusPolicy intentv1alpha1.NimbusPolicy
	err = r.Get(ctx, types.NamespacedName{Name: securityIntentBinding.Name, Namespace: securityIntentBinding.Namespace}, &nimbusPolicy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.createNimbusPolicy(ctx, nimbusPolicyToCreate)
		}
		logger.Error(err, "failed to fetch NimbusPolicy", "nimbusPolicy.name", securityIntentBinding.Name, "nimbusPolicy.namespace", securityIntentBinding.Namespace)
		return nil, err
	}

	return r.updateNimbusPolicy(ctx, &nimbusPolicy, nimbusPolicyToCreate)
}

func (r *SecurityIntentBindingReconciler) createNimbusPolicy(ctx context.Context, nimbusPolicyToCreate *intentv1alpha1.NimbusPolicy) (*intentv1alpha1.NimbusPolicy, error) {
	logger := log.FromContext(ctx)

	err := r.Create(ctx, nimbusPolicyToCreate)
	if err != nil {
		logger.Error(err, "failed to create NimbusPolicy", "nimbusPolicy.name", nimbusPolicyToCreate.Name, "nimbusPolicy.namespace", nimbusPolicyToCreate.Namespace)
		return nil, err
	}

	logger.V(2).Info("nimbusPolicy created", "nimbusPolicy.name", nimbusPolicyToCreate.Name, "nimbusPolicy.namespace", nimbusPolicyToCreate.Namespace)
	return nimbusPolicyToCreate, nil
}

func (r *SecurityIntentBindingReconciler) updateNimbusPolicy(ctx context.Context, existingNimbusPolicy *intentv1alpha1.NimbusPolicy, updatedNimbusPolicy *intentv1alpha1.NimbusPolicy) (*intentv1alpha1.NimbusPolicy, error) {
	logger := log.FromContext(ctx)

	existingNimbusPolicySpecBytes, _ := json.Marshal(existingNimbusPolicy.Spec)
	newNimbusPolicySpecBytes, _ := json.Marshal(updatedNimbusPolicy.Spec)
	if bytes.Equal(existingNimbusPolicySpecBytes, newNimbusPolicySpecBytes) {
		return existingNimbusPolicy, nil
	}

	updatedNimbusPolicy.ResourceVersion = existingNimbusPolicy.ResourceVersion
	err := r.Update(ctx, updatedNimbusPolicy)
	if err != nil {
		logger.Error(err, "failed to update NimbusPolicy", "nimbusPolicy.name", updatedNimbusPolicy.Name, "nimbusPolicy.namespace", updatedNimbusPolicy.Namespace)
		return nil, err
	}

	logger.V(2).Info("nimbusPolicy updated", "nimbusPolicy.name", updatedNimbusPolicy.Name, "nimbusPolicy.namespace", updatedNimbusPolicy.Namespace)
	return updatedNimbusPolicy, nil
}

func (r *SecurityIntentBindingReconciler) updateNpStatus(ctx context.Context, nimbusPolicy *intentv1alpha1.NimbusPolicy) error {
	nimbusPolicy.Status = intentv1alpha1.NimbusPolicyStatus{
		Status: StatusCreated,
	}
	return r.Status().Update(ctx, nimbusPolicy)
}

func (r *SecurityIntentBindingReconciler) updateSibStatusWithBoundNpAndSisInfo(ctx context.Context, existingSib *intentv1alpha1.SecurityIntentBinding, existingNp *intentv1alpha1.NimbusPolicy) error {
	existingSib.Status.Status = StatusCreated
	existingSib.Status.NimbusPolicy = existingNp.Name
	existingSib.Status.CountOfBoundIntents = int32(len(existingNp.Spec.NimbusRules))
	existingSib.Status.BoundIntents = r.getBoundIntents(ctx, existingSib.Spec.Intents)
	return r.Status().Update(ctx, existingSib)
}

func (r *SecurityIntentBindingReconciler) getBoundIntents(ctx context.Context, intents []intentv1alpha1.MatchIntent) []string {
	var boundIntentsName []string
	for _, intent := range intents {
		var currIntent intentv1alpha1.SecurityIntent
		if err := r.Get(ctx, types.NamespacedName{Name: intent.Name}, &currIntent); err != nil {
			continue
		}
		boundIntentsName = append(boundIntentsName, currIntent.Name)
	}
	return boundIntentsName
}

// findBindingsMatchingWithIntent finds SecurityIntentBindings that reference given SecurityIntent.
func (r *SecurityIntentBindingReconciler) findBindingsMatchingWithIntent(ctx context.Context, securityIntent client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	var requests []reconcile.Request

	var bindings intentv1alpha1.SecurityIntentBindingList
	if err := r.List(ctx, &bindings); err != nil {
		logger.Error(err, "failed to list SecurityIntentBinding")
		return requests
	}

	for _, binding := range bindings.Items {
		for _, intent := range binding.Spec.Intents {
			if intent.Name == securityIntent.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      binding.Name,
						Namespace: binding.Namespace,
					},
				})
				break
			}
		}
	}

	return requests
}

func (r *SecurityIntentBindingReconciler) deleteNimbusPolicyIfExists(ctx context.Context, name, namespace string) error {
	var nimbusPolicyToDelete intentv1alpha1.NimbusPolicy
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &nimbusPolicyToDelete); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if err := r.Delete(ctx, &nimbusPolicyToDelete); err != nil {
		return err
	}

	return nil
}

func (r *SecurityIntentBindingReconciler) removeNpAndSisDetailsFromSibStatus(ctx context.Context, bindingName, namespace string) error {
	var securityIntentBinding intentv1alpha1.SecurityIntentBinding
	if err := r.Get(ctx, types.NamespacedName{Name: bindingName, Namespace: namespace}, &securityIntentBinding); err != nil {
		return err
	}

	securityIntentBinding.Status.NimbusPolicy = ""
	securityIntentBinding.Status.CountOfBoundIntents = 0
	securityIntentBinding.Status.BoundIntents = []string{}

	return r.Status().Update(ctx, &securityIntentBinding)
}
