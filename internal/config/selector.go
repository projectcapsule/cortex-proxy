package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LabelSelector struct {
	MatchLabels      map[string]string          `yaml:"matchLabels,omitempty"`
	MatchExpressions []LabelSelectorRequirement `yaml:"matchExpressions,omitempty"`
}

type LabelSelectorRequirement struct {
	Key      string   `yaml:"key"`
	Operator string   `yaml:"operator"`
	Values   []string `yaml:"values,omitempty"`
}

func (l *LabelSelector) Selector() *metav1.LabelSelector {
	r := &metav1.LabelSelector{
		MatchLabels: l.MatchLabels,
	}
	for _, req := range l.MatchExpressions {
		r.MatchExpressions = append(r.MatchExpressions, metav1.LabelSelectorRequirement{
			Key:      req.Key,
			Operator: metav1.LabelSelectorOperator(req.Operator),
			Values:   req.Values,
		})
	}

	return r
}
