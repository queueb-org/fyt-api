package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"fyt.queueb.org/api/core/scheme"
)

// ValidateVersionedCreate converts a non-nil versioned object and runs its create
// validator. It returns field violations separately from conversion errors.
// The validator must treat the converted object as read-only because it may
// share data with the input.
func ValidateVersionedCreate[Internal any](object runtime.Object, validate func(*Internal) field.ErrorList) (field.ErrorList, error) {
	var internal Internal
	if err := scheme.Scheme.Convert(object, &internal, nil); err != nil {
		return nil, err
	}
	return validate(&internal), nil
}

// ValidateVersionedUpdate converts two non-nil versioned objects and passes the
// old object first and the new object second to validate. If either conversion
// fails, it returns the error without calling the validator. The validator must
// treat both converted objects as read-only because they may share input data.
func ValidateVersionedUpdate[Internal any](
	oldObject, newObject runtime.Object,
	validate func(*Internal, *Internal) field.ErrorList,
) (field.ErrorList, error) {
	var oldInternal, newInternal Internal
	if err := scheme.Scheme.Convert(oldObject, &oldInternal, nil); err != nil {
		return nil, fmt.Errorf("convert old object: %w", err)
	}
	if err := scheme.Scheme.Convert(newObject, &newInternal, nil); err != nil {
		return nil, fmt.Errorf("convert new object: %w", err)
	}
	return validate(&oldInternal, &newInternal), nil
}
