package validation

import (
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"fyt.queueb.org/api/core"
)

// OnCreateTimeEntry is the validation entry point for creating a [core.TimeEntry].
func OnCreateTimeEntry(entry *core.TimeEntry) (errorList field.ErrorList) {
	for _, verifier := range []func(entry *core.TimeEntry) (errorList field.ErrorList){
		validateTimeEntryStart,
		validateTimeEntryEnd,
		validateTimeEntryOrder,
		validateTimeEntryDescription,
		validateTimeEntryPrecision,
	} {
		errorList = append(errorList, verifier(entry)...)
	}
	return
}

// validateTimeEntryStart requires a start timestamp without changing its value.
func validateTimeEntryStart(entry *core.TimeEntry) field.ErrorList {
	errorList := field.ErrorList{}
	if entry.Start.IsZero() {
		errorList = append(errorList, field.Required(
			field.NewPath("start"),
			"must be specified",
		))
	}
	return errorList
}

// validateTimeEntryEnd requires an end timestamp without changing its value.
func validateTimeEntryEnd(entry *core.TimeEntry) field.ErrorList {
	errorList := field.ErrorList{}
	if entry.End.IsZero() {
		errorList = append(errorList, field.Required(
			field.NewPath("end"),
			"must be specified",
		))
	}
	return errorList
}

// validateTimeEntryOrder requires the end to be strictly after the start.
// Missing boundaries are checked separately.
func validateTimeEntryOrder(entry *core.TimeEntry) field.ErrorList {
	errorList := field.ErrorList{}
	if entry.Start.IsZero() || entry.End.IsZero() {
		return errorList
	}
	if !entry.End.Time.After(entry.Start.Time) {
		errorList = append(errorList, field.Invalid(
			field.NewPath("end"), entry.End.UTC().Format(time.RFC3339Nano),
			"must be strictly after start",
		))
	}
	return errorList
}

// validateTimeEntryDescription requires nonblank text without changing its value.
func validateTimeEntryDescription(entry *core.TimeEntry) field.ErrorList {
	errorList := field.ErrorList{}
	if strings.TrimSpace(entry.Description) == "" {
		errorList = append(errorList, field.Required(
			field.NewPath("description"),
			"must not be empty or contain only whitespace",
		))
	}
	return errorList
}

// validateTimeEntryPrecision rejects fractional seconds without changing values.
// Run before any operation that loses precision.
func validateTimeEntryPrecision(entry *core.TimeEntry) field.ErrorList {
	errorList := field.ErrorList{}
	if entry.Start.Nanosecond() != 0 {
		errorList = append(errorList, field.Invalid(
			field.NewPath("start"), entry.Start.UTC().Format(time.RFC3339Nano),
			"must not contain fractional seconds",
		))
	}
	if entry.End.Nanosecond() != 0 {
		errorList = append(errorList, field.Invalid(
			field.NewPath("end"), entry.End.UTC().Format(time.RFC3339Nano),
			"must not contain fractional seconds",
		))
	}
	return errorList
}

// OnUpdateTimeEntry validates the new [core.TimeEntry] using the creation rules.
// Update-specific validation rules will be added separately.
func OnUpdateTimeEntry(oldObj, newObj *core.TimeEntry) (errorList field.ErrorList) {
	errorList = append(errorList, OnCreateTimeEntry(newObj)...)
	return
}

// TimeEntryDeprecations returns warnings for deprecated [core.TimeEntry] fields.
// No deprecations are currently defined.
func TimeEntryDeprecations(obj *core.TimeEntry) (out []string) {
	return
}
