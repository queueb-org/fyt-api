package core

import (
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"queueb.org/apimachinery/pkg/conditions"
)

var _ conditions.Accessor = &TimeEntry{}

// SetConditions sets conditions
func (c *TimeEntry) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = slices.Clone(conditions)
}

// GetConditions returns resource conditions.
func (c *TimeEntry) GetConditions() *[]metav1.Condition {
	return &c.Status.Conditions
}
