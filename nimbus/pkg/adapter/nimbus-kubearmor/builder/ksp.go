package builder

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	kubearmorv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/api/security.kubearmor.com/v1"
	"sigs.k8s.io/yaml"

	intentv1alpha1 "github.com/anurag-rajawat/tutorials/nimbus/api/v1alpha1"
)

const (
	ActionEnforce  = "Enforce"
	ActionAudit    = "Audit"
	LabelPartOf    = "app.kubernetes.io/part-of"
	LabelManagedBy = "app.kubernetes.io/managed-by"
)

func pkgMgrPolicy(logger logr.Logger, np *intentv1alpha1.NimbusPolicy, rule intentv1alpha1.Rule) *kubearmorv1.KubeArmorPolicy {
	ksp, err := downloadKsp(
		"https://raw.githubusercontent.com/kubearmor/policy-templates/main/nist/system/ksp-nist-si-4-execute-package-management-process-in-container.yaml",
	)
	if err != nil {
		logger.Error(err, "failed to download KubeArmor policy")
		return nil
	}

	ksp.ObjectMeta.Name = np.Name + "-" + strings.ToLower(rule.ID)
	ksp.ObjectMeta.Namespace = np.Namespace
	ksp.Spec.Selector.MatchLabels = np.Spec.Selector.MatchLabels

	if ksp.Labels == nil {
		ksp.Labels = map[string]string{}
	}
	ksp.Labels[LabelPartOf] = np.Name + "-" + "nimbuspolicy"
	ksp.Labels[LabelManagedBy] = "nimbus-kubearmor"

	if rule.RuleAction == ActionEnforce {
		ksp.Spec.Process.Action = "Block"
	} else {
		ksp.Spec.Process.Action = ActionAudit
	}

	return ksp
}

func downloadKsp(fileUrl string) (*kubearmorv1.KubeArmorPolicy, error) {
	ksp := &kubearmorv1.KubeArmorPolicy{}
	response, err := http.Get(fileUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &ksp)
	return ksp, err
}
