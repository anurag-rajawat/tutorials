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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Rule defines a single rule within a NimbusPolicySpec
type Rule struct {
	// ID is a unique identifier for the rule, used by security engine adapters.
	ID string `json:"id"`

	// RuleAction specifies the action to be taken when the rule matches.
	RuleAction string `json:"action"`

	// Params is an optional map of parameters associated with the rule.
	Params map[string][]string `json:"params,omitempty"`
}

// NimbusPolicySpec defines the desired state of NimbusPolicy
type NimbusPolicySpec struct {
	// NimbusRules is a list of rules that define the policy.
	NimbusRules []Rule `json:"rules"`

	// Selector specifies the workload resources that the policy applies to.
	Selector WorkloadSelector `json:"selector"`
}

// NimbusPolicyStatus defines the observed state of NimbusPolicy
type NimbusPolicyStatus struct {
	Status                string   `json:"status"`
	GeneratedPoliciesName []string `json:"policiesName,omitempty"`
	CountOfPolicies       int32    `json:"policies,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=np
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Policies",type="integer",JSONPath=".status.policies"

// NimbusPolicy is the Schema for the nimbuspolicies API
type NimbusPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NimbusPolicySpec   `json:"spec,omitempty"`
	Status NimbusPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NimbusPolicyList contains a list of NimbusPolicy
type NimbusPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NimbusPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NimbusPolicy{}, &NimbusPolicyList{})
}
