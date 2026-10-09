package scheme

import (
	"testing"

	"fyt.queueb.org/api/core"
)

func TestDecodeTimeEntry(t *testing.T) {
	data := []byte(`{"apiVersion":"fyt.queueb.org/v1alpha1","kind":"TimeEntry","ticket":"FYT-1"}`)
	obj, _, err := Strict.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		t.Fatalf("could not decode time entry: %v", err)
	}
	entry, ok := obj.(*core.TimeEntry)
	if !ok {
		t.Fatalf("expected internal time entry, got %T", obj)
	}
	if entry.Ticket != "FYT-1" {
		t.Errorf("expected ticket FYT-1, got %q", entry.Ticket)
	}
}
