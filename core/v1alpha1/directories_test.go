package v1alpha1

import (
	"encoding/json"
	"reflect"
	"testing"

	"fyt.queueb.org/api/core"
	"sigs.k8s.io/yaml"
)

func TestDirectoryListsRoundTrip(t *testing.T) {
	scheme := WithNewLocalScheme(t)
	cfg := &Config{
		InputDirectories: []Directory{
			{Path: "./timesheets", Description: "Main records"},
			{Path: "./corrections", Description: "Corrections"},
		},
		Documents: DocumentsConfig{
			TemplatesDirectories: []Directory{
				{Path: "./templates", Description: "Custom templates"},
				{Path: "./shared/templates", Description: "Shared templates"},
			},
		},
	}
	for _, format := range []struct {
		name      string
		marshal   func(any) ([]byte, error)
		unmarshal func([]byte, any) error
	}{
		{name: "json", marshal: json.Marshal, unmarshal: json.Unmarshal},
		{
			name:    "yaml",
			marshal: yaml.Marshal,
			unmarshal: func(data []byte, out any) error {
				return yaml.UnmarshalStrict(data, out)
			},
		},
	} {
		t.Run(format.name, func(t *testing.T) {
			data, err := format.marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Config
			if err := format.unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(&decoded, cfg) {
				t.Fatal("directory paths or descriptions changed during serialization")
			}
			copy := decoded.DeepCopy()
			copy.InputDirectories[0].Path = "./other-input"
			copy.Documents.TemplatesDirectories[0].Description = "Other templates"
			if !reflect.DeepEqual(&decoded, cfg) {
				t.Fatal("mutating a versioned copy changed the original directory lists")
			}
			internal := &core.Config{}
			if err := scheme.Convert(&decoded, internal, nil); err != nil {
				t.Fatal(err)
			}
			internalCopy := internal.DeepCopy()
			internalCopy.InputDirectories[0].Path = "./other-input"
			internalCopy.Documents.TemplatesDirectories[0].Description = "Other templates"
			var roundTrip Config
			if err := scheme.Convert(internal, &roundTrip, nil); err != nil {
				t.Fatal(err)
			}
			inputsMatch := reflect.DeepEqual(roundTrip.InputDirectories, cfg.InputDirectories)
			templatesMatch := reflect.DeepEqual(roundTrip.Documents.TemplatesDirectories, cfg.Documents.TemplatesDirectories)
			if !inputsMatch || !templatesMatch {
				t.Fatal("directory lists changed during conversion or copying")
			}
		})
	}
}
