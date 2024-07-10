package builder

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	intentv1alpha1 "github.com/anurag-rajawat/tutorials/nimbus/api/v1alpha1"
	buildererrors "github.com/anurag-rajawat/tutorials/nimbus/pkg/utils/errors"
)

func BuildNimbusPolicy(ctx context.Context, k8sClient client.Client, securityIntentBinding intentv1alpha1.SecurityIntentBinding) (*intentv1alpha1.NimbusPolicy, error) {
	intents := extractIntents(ctx, k8sClient, &securityIntentBinding)
	if len(intents) == 0 {
		return nil, buildererrors.ErrSecurityIntentsNotFound
	}

	var nimbusRules []intentv1alpha1.Rule
	for _, intent := range intents {
		nimbusRules = append(nimbusRules, intentv1alpha1.Rule{
			ID:         intent.Spec.Intent.ID,
			RuleAction: intent.Spec.Intent.Action,
			Params:     intent.Spec.Intent.Params,
		})
	}

	nimbusPolicy := &intentv1alpha1.NimbusPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NimbusPolicy",
			APIVersion: intentv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      securityIntentBinding.Name,
			Namespace: securityIntentBinding.Namespace,
			Labels:    securityIntentBinding.Labels,
		},
		Spec: intentv1alpha1.NimbusPolicySpec{
			NimbusRules: nimbusRules,
			Selector:    securityIntentBinding.Spec.Selector,
		},
	}
	return nimbusPolicy, nil
}

func extractIntents(ctx context.Context, k8sClient client.Client, securityIntentBinding *intentv1alpha1.SecurityIntentBinding) []intentv1alpha1.SecurityIntent {
	var intentsToReturn []intentv1alpha1.SecurityIntent
	for _, intent := range securityIntentBinding.Spec.Intents {
		var currSecurityIntent intentv1alpha1.SecurityIntent
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: intent.Name}, &currSecurityIntent); err != nil {
			continue
		}
		intentsToReturn = append(intentsToReturn, currSecurityIntent)
	}
	return intentsToReturn
}
