package validation

import (
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"fyt.queueb.org/api/core"
)

func TestOnCreateTimeEntry(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		entry := &core.TimeEntry{
			Start:       metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC)),
			End:         metav1.NewTime(time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC)),
			Description: "Fix parser",
		}
		if errs := OnCreateTimeEntry(entry); len(errs) != 0 {
			t.Errorf("expected no validation errors, got %v", errs)
		}
	})

	t.Run("on-err", func(t *testing.T) {
		entry := &core.TimeEntry{}
		expected := field.ErrorList{
			field.Required(
				field.NewPath("start"),
				"must be specified",
			),
			field.Required(
				field.NewPath("end"),
				"must be specified",
			),
			field.Required(
				field.NewPath("description"),
				"must not be empty or contain only whitespace",
			),
		}
		if errs := OnCreateTimeEntry(entry); !reflect.DeepEqual(errs, expected) {
			t.Errorf("expected collected field errors %#v, got %#v", expected, errs)
		}
	})

	t.Run("invalid-order", func(t *testing.T) {
		entry := &core.TimeEntry{
			Start:       metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC)),
			Description: "Fix parser",
		}
		entry.End = entry.Start
		expected := field.ErrorList{
			field.Invalid(
				field.NewPath("end"), "2026-09-08T09:00:00Z",
				"must be strictly after start",
			),
		}
		if errs := OnCreateTimeEntry(entry); !reflect.DeepEqual(errs, expected) {
			t.Errorf("expected collected field errors %#v, got %#v", expected, errs)
		}
	})

	t.Run("fractional-boundaries", func(t *testing.T) {
		entry := &core.TimeEntry{
			Start:       metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 0, 1, time.UTC)),
			End:         metav1.NewTime(time.Date(2026, time.September, 8, 10, 0, 0, 999999999, time.UTC)),
			Description: "Fix parser",
		}
		expected := field.ErrorList{
			field.Invalid(
				field.NewPath("start"), "2026-09-08T09:00:00.000000001Z",
				"must not contain fractional seconds",
			),
			field.Invalid(
				field.NewPath("end"), "2026-09-08T10:00:00.999999999Z",
				"must not contain fractional seconds",
			),
		}
		if errs := OnCreateTimeEntry(entry); !reflect.DeepEqual(errs, expected) {
			t.Errorf("expected collected field errors %#v, got %#v", expected, errs)
		}
	})
}

func TestValidateTimeEntryStart(t *testing.T) {
	for _, test := range []struct {
		name     string
		start    metav1.Time
		expected field.ErrorList
	}{
		{
			name: "missing-start",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("start"),
					"must be specified",
				),
			},
		},
		{
			name:  "specified-start",
			start: metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 30, 0, time.UTC)),
		},
		{
			name:  "midnight",
			start: metav1.NewTime(time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "unix-epoch",
			start: metav1.NewTime(time.Unix(0, 0).UTC()),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := &core.TimeEntry{Start: test.start}
			entryBeforeValidation := entry.DeepCopy()
			errs := validateTimeEntryStart(entry)
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

func TestValidateTimeEntryEnd(t *testing.T) {
	for _, test := range []struct {
		name     string
		end      metav1.Time
		expected field.ErrorList
	}{
		{
			name: "missing-end",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("end"),
					"must be specified",
				),
			},
		},
		{
			name: "specified-end",
			end:  metav1.NewTime(time.Date(2026, time.September, 8, 10, 0, 30, 0, time.UTC)),
		},
		{
			name: "midnight",
			end:  metav1.NewTime(time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC)),
		},
		{
			name: "unix-epoch",
			end:  metav1.NewTime(time.Unix(0, 0).UTC()),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := &core.TimeEntry{End: test.end}
			entryBeforeValidation := entry.DeepCopy()
			errs := validateTimeEntryEnd(entry)
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

func TestValidateTimeEntryOrder(t *testing.T) {
	start := metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC))
	earlierEnd := metav1.NewTime(start.Add(-time.Second))
	equalEndInAnotherTimezone := metav1.NewTime(start.In(time.FixedZone("UTC+03", 3*60*60)))
	for _, test := range []struct {
		name     string
		start    metav1.Time
		end      metav1.Time
		expected field.ErrorList
	}{
		{
			name: "missing-boundaries",
		},
		{
			name: "missing-start",
			end:  start,
		},
		{
			name:  "missing-end",
			start: start,
		},
		{
			name:  "one-second-later",
			start: start,
			end:   metav1.NewTime(start.Add(time.Second)),
		},
		{
			name:  "across-midnight",
			start: metav1.NewTime(time.Date(2026, time.September, 8, 23, 59, 59, 0, time.UTC)),
			end:   metav1.NewTime(time.Date(2026, time.September, 9, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:  "equal-boundaries",
			start: start,
			end:   start,
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("end"), "2026-09-08T09:00:00Z",
					"must be strictly after start",
				),
			},
		},
		{
			name:  "end-before-start",
			start: start,
			end:   earlierEnd,
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("end"), "2026-09-08T08:59:59Z",
					"must be strictly after start",
				),
			},
		},
		{
			name:  "equal-instants-in-different-timezones",
			start: start,
			end:   equalEndInAnotherTimezone,
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("end"), "2026-09-08T09:00:00Z",
					"must be strictly after start",
				),
			},
		},
		{
			name:  "later-instant-with-earlier-local-time",
			start: start,
			end:   metav1.NewTime(time.Date(2026, time.September, 8, 7, 0, 0, 0, time.FixedZone("UTC-03", -3*60*60))),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := &core.TimeEntry{Start: test.start, End: test.end}
			entryBeforeValidation := entry.DeepCopy()
			errs := validateTimeEntryOrder(entry)
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

	t.Run("fractional-end-message", func(t *testing.T) {
		entry := &core.TimeEntry{
			Start: metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 1, 0, time.UTC)),
			End:   metav1.NewTime(time.Date(2026, time.September, 8, 12, 0, 0, 999999999, time.FixedZone("UTC+03", 3*60*60))),
		}
		errs := validateTimeEntryOrder(entry)
		if len(errs) != 1 {
			t.Fatalf("expected one ordering error, got %v", errs)
		}
		expectedMessage := `end: Invalid value: "2026-09-08T09:00:00.999999999Z": must be strictly after start`
		if actualMessage := errs[0].Error(); actualMessage != expectedMessage {
			t.Errorf("expected error message %q, got %q", expectedMessage, actualMessage)
		}
	})
}

func TestValidateTimeEntryDescription(t *testing.T) {
	requiredErrors := field.ErrorList{
		field.Required(
			field.NewPath("description"),
			"must not be empty or contain only whitespace",
		),
	}
	for _, test := range []struct {
		name        string
		description string
		expected    field.ErrorList
	}{
		{
			name:     "empty-description",
			expected: requiredErrors,
		},
		{
			name:        "spaces-only",
			description: "   ",
			expected:    requiredErrors,
		},
		{
			name:        "ascii-whitespace-only",
			description: " \t\r\n\v\f",
			expected:    requiredErrors,
		},
		{
			name:        "unicode-whitespace-only",
			description: "\u00a0\u2003\u3000",
			expected:    requiredErrors,
		},
		{
			name:        "nonblank-description",
			description: "Fix parser",
		},
		{
			name:        "single-nonblank-character",
			description: "x",
		},
		{
			name:        "surrounding-whitespace",
			description: "\t Fix parser \u00a0",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := &core.TimeEntry{Description: test.description}
			entryBeforeValidation := entry.DeepCopy()
			errs := validateTimeEntryDescription(entry)
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

func TestValidateTimeEntryPrecision(t *testing.T) {
	start := metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC))
	end := metav1.NewTime(time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC))
	fractionalStart := metav1.NewTime(start.Add(time.Nanosecond))
	fractionalEnd := metav1.NewTime(end.Add(999999999 * time.Nanosecond))
	startErrors := field.ErrorList{
		field.Invalid(
			field.NewPath("start"), "2026-09-08T09:00:00.000000001Z",
			"must not contain fractional seconds",
		),
	}
	endErrors := field.ErrorList{
		field.Invalid(
			field.NewPath("end"), "2026-09-08T10:00:00.999999999Z",
			"must not contain fractional seconds",
		),
	}
	for _, test := range []struct {
		name     string
		start    metav1.Time
		end      metav1.Time
		expected field.ErrorList
	}{
		{
			name: "missing-boundaries",
		},
		{
			name:  "whole-seconds",
			start: start,
			end:   end,
		},
		{
			name:     "fractional-start",
			start:    fractionalStart,
			end:      end,
			expected: startErrors,
		},
		{
			name:     "fractional-end",
			start:    start,
			end:      fractionalEnd,
			expected: endErrors,
		},
		{
			name:     "fractional-boundaries",
			start:    fractionalStart,
			end:      fractionalEnd,
			expected: field.ErrorList{startErrors[0], endErrors[0]},
		},
		{
			name:     "fractional-start-with-missing-end",
			start:    fractionalStart,
			expected: startErrors,
		},
		{
			name:     "fractional-end-with-missing-start",
			end:      fractionalEnd,
			expected: endErrors,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := &core.TimeEntry{Start: test.start, End: test.end}
			entryBeforeValidation := entry.DeepCopy()
			errs := validateTimeEntryPrecision(entry)
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

func TestOnUpdateTimeEntry(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		oldEntry := &core.TimeEntry{}
		newEntry := &core.TimeEntry{
			Start:       metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC)),
			End:         metav1.NewTime(time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC)),
			Description: "Fix parser",
		}
		if errs := OnUpdateTimeEntry(oldEntry, newEntry); len(errs) != 0 {
			t.Errorf("expected no validation errors for the new time entry, got %v", errs)
		}
	})

	t.Run("on-err", func(t *testing.T) {
		oldEntry := &core.TimeEntry{
			Start:       metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC)),
			End:         metav1.NewTime(time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC)),
			Description: "Fix parser",
		}
		newEntry := &core.TimeEntry{
			Start: metav1.NewTime(time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC)),
			End:   metav1.NewTime(time.Date(2026, time.September, 8, 10, 0, 0, 0, time.UTC)),
		}
		if errs := OnUpdateTimeEntry(oldEntry, newEntry); len(errs) == 0 {
			t.Error("expected creation validation errors for the new time entry")
		}
	})
}

func TestTimeEntryDeprecations(t *testing.T) {
	t.Run("no-deprecations", func(t *testing.T) {
		entry := &core.TimeEntry{}
		if warnings := TimeEntryDeprecations(entry); len(warnings) != 0 {
			t.Errorf("expected no deprecation warnings, got %v", warnings)
		}
	})
}
