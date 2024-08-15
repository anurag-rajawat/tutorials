package builder

import (
	"github.com/go-logr/logr"
	kubearmorv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/api/security.kubearmor.com/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	intentv1alpha1 "github.com/anurag-rajawat/tutorials/nimbus/api/v1alpha1"
	"github.com/anurag-rajawat/tutorials/nimbus/pkg/adapter/idpool"
)

func BuildPolicy(logger logr.Logger, scheme *runtime.Scheme, nimbusPolicy *intentv1alpha1.NimbusPolicy) []kubearmorv1.KubeArmorPolicy {
	var ksps []kubearmorv1.KubeArmorPolicy
	for _, nimbusRule := range nimbusPolicy.Spec.NimbusRules {
		if !idpool.IsSupportedId(nimbusRule.ID, "kubearmor") {
			logger.Info("KubeArmor adapter doesn't support this ID", "ID", nimbusRule.ID)
			continue
		}
		actualKsps := buildPolicy(logger, nimbusPolicy, nimbusRule, scheme)
		ksps = append(ksps, actualKsps...)
	}
	return ksps
}

func buildPolicy(logger logr.Logger, np *intentv1alpha1.NimbusPolicy, rule intentv1alpha1.Rule, scheme *runtime.Scheme) []kubearmorv1.KubeArmorPolicy {
	switch rule.ID {
	case idpool.PkgManagerExecution:
		ksp := pkgMgrPolicy(logger, np, rule)
		//Set NimbusPolicy as the owner of the KSP
		if err := ctrl.SetControllerReference(np, ksp, scheme); err != nil {
			logger.Error(err, "Failed to set controller reference on policy")
		}
		return []kubearmorv1.KubeArmorPolicy{*ksp}
	default:
		return nil
	}
}
