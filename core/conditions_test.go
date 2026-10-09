package core

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"queueb.org/apimachinery/pkg/conditions"
)

var (
	testConditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 0,
			LastTransitionTime: metav1.Time{},
			Reason:             "any reason",
			Message:            "any message",
		},
	}
)

func TestConditions(t *testing.T) {
	for _, make := range []func() conditions.Accessor{
		func() conditions.Accessor { return &TimeEntry{} },
	} {
		t.Run("set-get-conditions", func(in *testing.T) {
			resource := make()
			conditions.Set(resource, testConditions)
			expected := testConditions[:]
			if result := conditions.Get(resource); !reflect.DeepEqual(*result, expected) {
				in.Errorf("\nexp:\n%#v\ngot:\n%#v", expected, result)
			}
		})

		t.Run("add-condition", func(in *testing.T) {
			resource := make()
			for range 3 {
				conditions.Add(resource, &testConditions[0])
			}
			expected := testConditions

			if result := conditions.Get(resource); !reflect.DeepEqual(*result, expected) {
				in.Errorf("\nexp:\n%#v\ngot:\n%#v", expected, result)
			}
		})

		t.Run("add-or-update-condition", func(in *testing.T) {
			resource := make()
			for range 3 {
				conditions.AddOrUpdate(resource, &testConditions[0])
			}
			condition := testConditions[0]
			condition.Reason = "new reason"
			conditions.AddOrUpdate(resource, &condition)
			expected := []metav1.Condition{condition}

			if result := conditions.Get(resource); !reflect.DeepEqual(*result, expected) {
				in.Errorf("\nexp:\n%#v\ngot:\n%#v", expected, result)
			}
		})

		t.Run("update", func(in *testing.T) {
			resource := make()

			conditions.AddOrUpdate(resource, &testConditions[0])

			condition := testConditions[0]
			condition.Reason = "new reason"
			conditions.Update(resource, &condition)
			expected := []metav1.Condition{condition}

			if result := conditions.Get(resource); !reflect.DeepEqual(*result, expected) {
				in.Errorf("\nexp:\n%#v\ngot:\n%#v", expected, result)
			}

			// update should not add new condition
			conditions.Update(resource, &metav1.Condition{Type: "NewConditionType"})
			if result := conditions.Get(resource); !reflect.DeepEqual(*result, expected) {
				in.Errorf("\nexp:\n%#v\ngot:\n%#v", expected, result)
			}
		})
	}
}
