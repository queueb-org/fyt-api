package validation

import (
	"reflect"
	"testing"
	"time"

	"common.queueb.org/tests"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"fyt.queueb.org/api/core"
	"fyt.queueb.org/api/core/scheme"
)

// TestIntegration_TimeEntry_Precision checks strict JSON/YAML decoding, conversion
// to core.TimeEntry, and validation together: represented nonzero fractions are
// rejected, while explicit zero fractions and the accepted subnanosecond exception
// are read as whole seconds (TE-005).
func TestIntegration_TimeEntry_Precision(t *testing.T) {
	start := time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name          string
		path          string
		expectedStart time.Time
		expectedEnd   time.Time
		expected      field.ErrorList
	}{
		{
			name:          "json-fractional-seconds",
			path:          testJSONFractionalEntry,
			expectedStart: start.Add(time.Nanosecond),
			expectedEnd:   end.Add(999999999 * time.Nanosecond),
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("start"), "2026-09-08T09:00:00.000000001Z",
					"must not contain fractional seconds",
				),
				field.Invalid(
					field.NewPath("end"), "2026-09-08T10:00:00.999999999Z",
					"must not contain fractional seconds",
				),
			},
		},
		{
			name:          "json-ten-nanoseconds",
			path:          testJSONTenNanosecondEntry,
			expectedStart: start.Add(10 * time.Nanosecond),
			expectedEnd:   end.Add(10 * time.Nanosecond),
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("start"), "2026-09-08T09:00:00.00000001Z",
					"must not contain fractional seconds",
				),
				field.Invalid(
					field.NewPath("end"), "2026-09-08T10:00:00.00000001Z",
					"must not contain fractional seconds",
				),
			},
		},
		{
			name:          "json-zero-fraction",
			path:          testJSONZeroFractionEntry,
			expectedStart: start,
			expectedEnd:   end,
		},
		{
			name:          "json-subnanosecond-exception",
			path:          testJSONSubnanosecondEntry,
			expectedStart: start,
			expectedEnd:   end,
		},
		{
			name:          "yaml-fractional-seconds",
			path:          testYAMLFractionalEntry,
			expectedStart: start.Add(time.Nanosecond),
			expectedEnd:   end.Add(999999999 * time.Nanosecond),
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("start"), "2026-09-08T09:00:00.000000001Z",
					"must not contain fractional seconds",
				),
				field.Invalid(
					field.NewPath("end"), "2026-09-08T10:00:00.999999999Z",
					"must not contain fractional seconds",
				),
			},
		},
		{
			name:          "yaml-ten-nanoseconds",
			path:          testYAMLTenNanosecondEntry,
			expectedStart: start.Add(10 * time.Nanosecond),
			expectedEnd:   end.Add(10 * time.Nanosecond),
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("start"), "2026-09-08T09:00:00.00000001Z",
					"must not contain fractional seconds",
				),
				field.Invalid(
					field.NewPath("end"), "2026-09-08T10:00:00.00000001Z",
					"must not contain fractional seconds",
				),
			},
		},
		{
			name:          "yaml-zero-fraction",
			path:          testYAMLZeroFractionEntry,
			expectedStart: start,
			expectedEnd:   end,
		},
		{
			name:          "yaml-subnanosecond-exception",
			path:          testYAMLSubnanosecondEntry,
			expectedStart: start,
			expectedEnd:   end,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := tests.WithReadFile(t, test.path)
			obj, _, err := scheme.Strict.UniversalDecoder().Decode(data, nil, nil)
			if err != nil {
				t.Fatalf("could not decode time entry: %v", err)
			}
			entry, ok := obj.(*core.TimeEntry)
			if !ok {
				t.Fatalf("expected internal time entry, got %T", obj)
			}
			if !entry.Start.Time.Equal(test.expectedStart) {
				t.Errorf("expected start %s, got %s", test.expectedStart.Format(time.RFC3339Nano), entry.Start.Format(time.RFC3339Nano))
			}
			if !entry.End.Time.Equal(test.expectedEnd) {
				t.Errorf("expected end %s, got %s", test.expectedEnd.Format(time.RFC3339Nano), entry.End.Format(time.RFC3339Nano))
			}
			entryBeforeValidation := entry.DeepCopy()
			errs := OnCreateTimeEntry(entry)
			if len(errs) != len(test.expected) {
				t.Fatalf("expected errors %v, got %v", test.expected, errs)
			}
			if len(errs) != 0 && !reflect.DeepEqual(errs, test.expected) {
				t.Errorf("expected errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(entry, entryBeforeValidation) {
				t.Error("validation changed the time entry")
			}
		})
	}
}

// TestIntegration_TimeEntryList_Precision checks strict JSON/YAML decoding and
// conversion of TimeEntryList items, followed by explicit item validation:
// .0000000001 is accepted as a whole second under the TE-005 exception, while
// .00000001 survives as 10 nanoseconds and produces errors for both boundaries.
func TestIntegration_TimeEntryList_Precision(t *testing.T) {
	start := time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name          string
		path          string
		expectedStart time.Time
		expectedEnd   time.Time
		expected      field.ErrorList
	}{
		{
			name:          "json-subnanosecond-exception",
			path:          testJSONSubnanosecondList,
			expectedStart: start,
			expectedEnd:   end,
		},
		{
			name:          "json-ten-nanoseconds",
			path:          testJSONTenNanosecondList,
			expectedStart: start.Add(10 * time.Nanosecond),
			expectedEnd:   end.Add(10 * time.Nanosecond),
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("start"), "2026-09-08T09:00:00.00000001Z",
					"must not contain fractional seconds",
				),
				field.Invalid(
					field.NewPath("end"), "2026-09-08T10:00:00.00000001Z",
					"must not contain fractional seconds",
				),
			},
		},
		{
			name:          "yaml-subnanosecond-exception",
			path:          testYAMLSubnanosecondList,
			expectedStart: start,
			expectedEnd:   end,
		},
		{
			name:          "yaml-ten-nanoseconds",
			path:          testYAMLTenNanosecondList,
			expectedStart: start.Add(10 * time.Nanosecond),
			expectedEnd:   end.Add(10 * time.Nanosecond),
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("start"), "2026-09-08T09:00:00.00000001Z",
					"must not contain fractional seconds",
				),
				field.Invalid(
					field.NewPath("end"), "2026-09-08T10:00:00.00000001Z",
					"must not contain fractional seconds",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := tests.WithReadFile(t, test.path)
			obj, _, err := scheme.Strict.UniversalDecoder().Decode(data, nil, nil)
			if err != nil {
				t.Fatalf("could not decode time entry list: %v", err)
			}
			list, ok := obj.(*core.TimeEntryList)
			if !ok {
				t.Fatalf("expected internal time entry list, got %T", obj)
			}
			if len(list.Items) != 1 {
				t.Fatalf("expected one item in list, got %d", len(list.Items))
			}
			listBeforeValidation := list.DeepCopy()
			entry := &list.Items[0]
			if !entry.Start.Time.Equal(test.expectedStart) {
				t.Errorf("expected item start %s, got %s", test.expectedStart.Format(time.RFC3339Nano), entry.Start.Format(time.RFC3339Nano))
			}
			if !entry.End.Time.Equal(test.expectedEnd) {
				t.Errorf("expected item end %s, got %s", test.expectedEnd.Format(time.RFC3339Nano), entry.End.Format(time.RFC3339Nano))
			}
			errs := OnCreateTimeEntry(entry)
			if len(errs) != len(test.expected) {
				t.Fatalf("expected item errors %v, got %v", test.expected, errs)
			}
			if len(errs) != 0 && !reflect.DeepEqual(errs, test.expected) {
				t.Errorf("expected item errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(list, listBeforeValidation) {
				t.Error("validation changed the time entry list")
			}
		})
	}
}
