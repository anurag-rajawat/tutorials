package manager

import (
	"context"
	"slices"
	"strings"

	kubearmorv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/api/security.kubearmor.com/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/anurag-rajawat/tutorials/nimbus/adapter/nimbus-kubearmor/builder"
	intentv1alpha1 "github.com/anurag-rajawat/tutorials/nimbus/api/v1alpha1"
)

func (m *manager) managePolicies(ctx context.Context, nimbusPolicyChan chan *intentv1alpha1.NimbusPolicy, kspChan chan *kubearmorv1.KubeArmorPolicy) {
	for {
		select {
		case <-ctx.Done():
			return
		case nimbusPolicy := <-nimbusPolicyChan:
			m.createOrUpdatePolicies(ctx, nimbusPolicy)
		case ksp := <-kspChan:
			m.reconcileKsp(ctx, ksp.Name, ksp.Namespace)
		}
	}
}

func (m *manager) reconcileKsp(ctx context.Context, name, namespace string) {
	npName := extractNpName(name)
	nimbusPolicy := &intentv1alpha1.NimbusPolicy{}
	if err := m.k8sClient.Get(ctx, types.NamespacedName{Name: npName, Namespace: namespace}, nimbusPolicy); err != nil {
		m.logger.Error(err, "failed to get NimbusPolicy", "nimbusPolicy.Name", nimbusPolicy.Name, "nimbusPolicy.Namespace", namespace)
		return
	}
	m.createOrUpdatePolicies(ctx, nimbusPolicy)
}

func extractNpName(name string) string {
	words := strings.Split(name, "-")
	return strings.Join(words[:len(words)-1], "-")
}

func (m *manager) createOrUpdatePolicies(ctx context.Context, nimbusPolicy *intentv1alpha1.NimbusPolicy) {
	ksps := builder.BuildPolicy(m.logger, m.scheme, nimbusPolicy)
	// Iterate using a separate index variable to avoid aliasing
	for idx := range ksps {
		ksp := &ksps[idx]

		var modified bool
		var existingKsp kubearmorv1.KubeArmorPolicy
		err := m.k8sClient.Get(ctx, types.NamespacedName{Name: ksp.Name, Namespace: ksp.Namespace}, &existingKsp)
		if err != nil && !apierrors.IsNotFound(err) {
			m.logger.Error(err, "failed to get existing KubeArmorPolicy", "kubeArmorPolicy.Name", nimbusPolicy.Name, "KubeArmorPolicy.Namespace", nimbusPolicy.Namespace)
			return
		}

		if err != nil {
			if apierrors.IsNotFound(err) {
				if err = m.k8sClient.Create(ctx, ksp); err != nil {
					m.logger.Error(err, "failed to create KubeArmorPolicy", "kubeArmorPolicy.Name", ksp.Name, "kubeArmorPolicy.Namespace", ksp.Namespace)
					return
				}
				modified = true
				m.logger.Info("Successfully created KubeArmorPolicy", "kubeArmorPolicy.Name", ksp.Name, "kubeArmorPolicy.Namespace", ksp.Namespace)
			}
		} else {
			ksp.ObjectMeta.ResourceVersion = existingKsp.ObjectMeta.ResourceVersion
			if err = m.k8sClient.Update(ctx, ksp); err != nil {
				m.logger.Error(err, "failed to configure existing KubeArmorPolicy", "kubeArmorPolicy.Name", ksp.Name, "kubeArmorPolicy.Namespace", ksp.Namespace)
				return
			}
			modified = true
			m.logger.Info("Configured KubeArmorPolicy", "kubeArmorPolicy.Name", ksp.Name, "kubeArmorPolicy.Namespace", ksp.Namespace)
		}

		if modified {
			var latestNimbusPolicy intentv1alpha1.NimbusPolicy
			if err = m.k8sClient.Get(ctx, types.NamespacedName{Name: nimbusPolicy.Name, Namespace: nimbusPolicy.Namespace}, &latestNimbusPolicy); err != nil {
				m.logger.Error(err, "failed to get existing NimbusPolicy", "nimbusPolicy.name", nimbusPolicy.Name, "nimbusPolicy.Namespace", nimbusPolicy.Namespace)
				return
			}

			if !slices.Contains(latestNimbusPolicy.Status.GeneratedPoliciesName, ksp.Name) {
				latestNimbusPolicy.Status.CountOfPolicies++
				latestNimbusPolicy.Status.GeneratedPoliciesName = append(latestNimbusPolicy.Status.GeneratedPoliciesName, ksp.Name)
				if err = m.k8sClient.Status().Update(ctx, &latestNimbusPolicy); err != nil {
					m.logger.Error(err, "failed to update KubeArmorPolicies info in NimbusPolicy")
				}
			}
		}
	}
}
