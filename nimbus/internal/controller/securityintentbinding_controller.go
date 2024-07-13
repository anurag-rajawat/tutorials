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
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

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

	if err = r.createOrUpdateNimbusPolicy(ctx, securityIntentBinding); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecurityIntentBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&intentv1alpha1.SecurityIntentBinding{}).
		Owns(&intentv1alpha1.NimbusPolicy{}).
		Complete(r)
}

func (r *SecurityIntentBindingReconciler) createOrUpdateNimbusPolicy(ctx context.Context, securityIntentBinding intentv1alpha1.SecurityIntentBinding) error {
	logger := log.FromContext(ctx)

	nimbusPolicyToCreate, err := builder.BuildNimbusPolicy(ctx, r.Client, securityIntentBinding)
	if err != nil {
		if errors.Is(err, buildererrors.ErrSecurityIntentsNotFound) {
			logger.Info("aborted NimbusPolicy creation, since no SecurityIntents were found")
			return nil
		}
		return err
	}

	var nimbusPolicy intentv1alpha1.NimbusPolicy
	err = r.Get(ctx, types.NamespacedName{Name: securityIntentBinding.Name, Namespace: securityIntentBinding.Namespace}, &nimbusPolicy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.createNimbusPolicy(ctx, nimbusPolicyToCreate)
		}
		logger.Error(err, "failed to fetch NimbusPolicy", "nimbusPolicy.name", securityIntentBinding.Name, "nimbusPolicy.namespace", securityIntentBinding.Namespace)
		return err
	}

	return r.updateNimbusPolicy(ctx, nimbusPolicy, nimbusPolicyToCreate)
}

func (r *SecurityIntentBindingReconciler) createNimbusPolicy(ctx context.Context, nimbusPolicyToCreate *intentv1alpha1.NimbusPolicy) error {
	logger := log.FromContext(ctx)

	err := r.Create(ctx, nimbusPolicyToCreate)
	if err != nil {
		logger.Error(err, "failed to create NimbusPolicy", "nimbusPolicy.name", nimbusPolicyToCreate.Name, "nimbusPolicy.namespace", nimbusPolicyToCreate.Namespace)
		return err
	}

	return nil
}

func (r *SecurityIntentBindingReconciler) updateNimbusPolicy(ctx context.Context, existingNimbusPolicy intentv1alpha1.NimbusPolicy, nimbusPolicyToCreate *intentv1alpha1.NimbusPolicy) error {
	logger := log.FromContext(ctx)

	nimbusPolicyToCreate.ResourceVersion = existingNimbusPolicy.ResourceVersion
	err := r.Update(ctx, nimbusPolicyToCreate)
	if err != nil {
		logger.Error(err, "failed to update NimbusPolicy", "nimbusPolicy.name", nimbusPolicyToCreate.Name, "nimbusPolicy.namespace", nimbusPolicyToCreate.Namespace)
		return err
	}

	return nil
}
