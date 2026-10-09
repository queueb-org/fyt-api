package scheme

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"common.queueb.org/tests"

	"fyt.queueb.org/api/core/v1alpha1"
)

const (
	validConfig = `apiVersion: fyt.queueb.org/v1alpha1
kind: Config
`
	testValidConfigLocation = "../../resources/config.example.yaml"
	testValidStatusLocation = "testdata/unexpected-status.yaml"

	invoiceConfigHeader     = "apiVersion: fyt.queueb.org/v1alpha1\nkind: InvoiceConfig\n"
	invoiceConfigListHeader = "apiVersion: fyt.queueb.org/v1alpha1\nkind: InvoiceConfigList\n"
)

func TestDecode(t *testing.T) {
	t.Run("ok", func(in *testing.T) {
		config, err := Decode(tests.WithOpenFile(in, testValidConfigLocation))
		if err != nil {
			in.Fatalf("Decode() error = %v", err)
		}
		if config == nil {
			in.Fatal("Decode() returned a nil config")
		}
	})

	t.Run("currency-before-validation", func(t *testing.T) {
		for _, test := range []struct {
			name, invoice, want string
		}{
			{name: "omitted", invoice: "{}", want: "USD"},
			{name: "null", invoice: "{currency: null}", want: "USD"},
			{name: "empty", invoice: "{currency: ''}"},
			{name: "whitespace", invoice: "{currency: '  '}", want: "  "},
		} {
			t.Run(test.name, func(t *testing.T) {
				raw := validConfig + "invoice: " + test.invoice + "\n"
				config, err := Decode(strings.NewReader(raw))
				if err != nil {
					t.Fatal(err)
				}
				if config.Invoice.Currency == nil || *config.Invoice.Currency != test.want {
					t.Fatalf("want currency %q, got %#v", test.want, config.Invoice.Currency)
				}
			})
		}
	})

	t.Run("cant-read", func(in *testing.T) {
		fd, err := os.Open(testValidConfigLocation)
		if err != nil {
			in.Fatalf("could not read test configuration file: %v", err)
		}

		// explicitly closing
		_ = fd.Close()
		if _, err := Decode(fd); !errors.Is(err, ErrDecode) {
			in.Errorf("expected: `%v`, got: `%v`", ErrDecode, err)
		}
	})

	t.Run("unknown-field", func(in *testing.T) {
		raw := tests.WithReadFileString(in, testValidConfigLocation) + "\nunknown: value\n"

		_, err := Decode(strings.NewReader(raw))
		if err == nil {
			in.Fatal("Decode() error = nil, want a strict decoding error")
		}
		if !strings.Contains(err.Error(), "unknown field \"unknown\"") {
			in.Fatalf("Decode() error = %q, want an unknown field error", err)
		}
	})

	t.Run("unsupported-version", func(in *testing.T) {
		data := `apiVersion: gena.apps.queueb.org/v1beta1
kind: Config`

		_, err := Decode(strings.NewReader(data))
		if err == nil {
			in.Fatal("Decode() error = nil, want a version error")
		}

		if !strings.Contains(err.Error(),
			"no kind \"Config\" is registered for version") {
			in.Errorf("Decode() error = %q, want an unregistered version error", err)
		}
	})

	t.Run("unsupported-api-object", func(in *testing.T) {
		data := `apiVersion: v1
kind: Status
status: Failure`

		_, err := Decode(strings.NewReader(data))
		if err == nil {
			in.Fatal("Decode() error = nil, want a version error")
		}
		if !strings.Contains(err.Error(),
			"could not decode object: unexpected object /v1, Kind=Status") {
			in.Fatalf("Decode() error = %q, want an unregistered version error", err)
		}
	})

	t.Run("duplicate-field", func(in *testing.T) {
		data := validConfig + "kind: Config\n"

		_, err := Decode(strings.NewReader(data))
		if err == nil {
			in.Fatal("Decode() error = nil, want a duplicate field error")
		}
		if !strings.Contains(err.Error(), "key \"kind\" already set") {
			in.Fatalf("Decode() error = %q, want a duplicate field error", err)
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("ok", func(in *testing.T) {
		config, err := Load(testValidConfigLocation)
		if err != nil {
			in.Errorf("Load() error = %v", err)
		}
		if config == nil {
			in.Errorf("Load() returned a nil config")
		}
	})

	t.Run("no-file-exists", func(in *testing.T) {
		if _, err := Load("non existent"); !errors.Is(err, ErrLoad) {
			in.Errorf("expected: `%v`, got: `%v`", ErrLoad, err)
		}
	})

	t.Run("non-config-obj", func(in *testing.T) {
		if _, err := Load(testValidStatusLocation); !errors.Is(err, ErrLoad) {
			in.Errorf("expected: `%v`, got: `%v`", ErrLoad, err)
		}
	})
}

func TestDecodeVersionedConfig(t *testing.T) {
	t.Run("defaults-and-explicit-values", func(t *testing.T) {
		raw := validConfig + "invoice:\n  monetaryPrecision: 0\nprocessing:\n  duplicatePolicy: Error\n"
		config, err := DecodeVersionedConfig(strings.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		if config.APIVersion != "fyt.queueb.org/v1alpha1" || config.Kind != "Config" {
			t.Errorf("unexpected type metadata: %+v", config.TypeMeta)
		}
		name := config.Reports.MemberName
		if name == nil {
			t.Fatal("expected a default member name, got nil")
		}
		if *name != v1alpha1.DefaultMemberName {
			t.Errorf("expected the default member name, got %q", *name)
		}
		if config.Invoice.MonetaryPrecision == nil || *config.Invoice.MonetaryPrecision != 0 {
			t.Errorf("expected explicit zero monetary precision, got %+v", config.Invoice)
		}
		if config.Processing.DuplicatePolicy != v1alpha1.DuplicatePolicyError {
			t.Errorf("expected the explicit duplicate policy, got %q", config.Processing.DuplicatePolicy)
		}
	})

	t.Run("unreadable", func(t *testing.T) {
		config, err := DecodeVersionedConfig(iotest.ErrReader(io.ErrUnexpectedEOF))
		if config != nil || !errors.Is(err, ErrDecode) {
			t.Fatalf("expected no config and a decode error, got %v, %v", config, err)
		}
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("expected the original read error, got %v", err)
		}
	})

	for _, test := range []struct {
		name string
		data string
		want string
	}{
		{"unknown-field", validConfig + "unknown: value\n", `unknown field "unknown"`},
		{"duplicate-key", validConfig + "kind: Config\n", `key "kind" already set`},
		{"wrong-kind", "apiVersion: fyt.queueb.org/v1alpha1\nkind: TimeEntryList\n", "unexpected object"},
		{"unsupported-version", "apiVersion: fyt.queueb.org/v9\nkind: Config\n", "fyt.queueb.org/v9"},
		{"missing-version", "kind: Config\n", "apiVersion"},
		{"missing-kind", "apiVersion: fyt.queueb.org/v1alpha1\n", "Kind"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, err := DecodeVersionedConfig(strings.NewReader(test.data))
			if config != nil || !errors.Is(err, ErrDecode) {
				t.Fatalf("expected no config and a decode error, got %v, %v", config, err)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("expected error containing %q, got %v", test.want, err)
			}
		})
	}

	t.Run("non-strict-option", func(t *testing.T) {
		config, err := DecodeVersionedConfig(strings.NewReader(validConfig+"unknown: value\n"), OptNoStrict())
		if config == nil || err != nil {
			t.Errorf("expected a config in non-strict mode, got %v, %v", config, err)
		}
	})
}

func TestDecodeObject(t *testing.T) {
	decoder := Strict.UniversalDeserializer()
	config, err := decodeObject[*v1alpha1.Config]([]byte(validConfig), decoder, "")
	if err != nil || config == nil {
		t.Fatalf("expected a Config, got %+v, %v", config, err)
	}

	for _, tc := range []struct {
		name   string
		data   string
		prefix string
		want   string
	}{
		{
			name:   "decode-error",
			data:   "kind: Config\n",
			prefix: "could not read configuration: ",
			want:   "could not decode object: could not read configuration: ",
		},
		{
			name: "wrong-object",
			data: `{"apiVersion":"fyt.queueb.org/v1alpha1","kind":"TimeEntry"}`,
			want: "could not decode object: unexpected object fyt.queueb.org/v1alpha1, Kind=TimeEntry",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, err := decodeObject[*v1alpha1.Config]([]byte(tc.data), decoder, tc.prefix)
			if config != nil || !errors.Is(err, ErrDecode) {
				t.Fatalf("expected a decode error, got %+v, %v", config, err)
			}
			if !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("expected error starting with %q, got %v", tc.want, err)
			}
		})
	}
}

func TestLoadVersionedConfig(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		config, err := LoadVersionedConfig(testValidConfigLocation)
		if config == nil || err != nil {
			t.Fatalf("expected a config, got %v, %v", config, err)
		}
		if config.Kind != "Config" {
			t.Errorf("expected Config, got %q", config.Kind)
		}
	})
	t.Run("missing-file", func(t *testing.T) {
		config, err := LoadVersionedConfig(filepath.Join(t.TempDir(), "missing.yaml"))
		if config != nil || !errors.Is(err, ErrLoad) {
			t.Fatalf("expected no config and a load error, got %v, %v", config, err)
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected a missing file error, got %v", err)
		}
	})
	t.Run("invalid-document", func(t *testing.T) {
		config, err := LoadVersionedConfig(testValidStatusLocation)
		if config != nil || !errors.Is(err, ErrLoad) {
			t.Fatalf("expected no config and a load error, got %v, %v", config, err)
		}
		if !errors.Is(err, ErrDecode) {
			t.Errorf("expected a decoding error, got %v", err)
		}
	})
}

func TestDecodeVersionedInvoiceConfig(t *testing.T) {
	const payload = `invoice:
  customer:
    email: billing@example.org
  tax: {}
  monetaryPrecision: 0
  additionalLines: []
`
	config, err := DecodeVersionedInvoiceConfig(strings.NewReader(invoiceConfigHeader + payload))
	if err != nil {
		t.Fatal(err)
	}
	want := v1alpha1.InvoiceSettingsOverride{
		Customer:          &v1alpha1.InvoicePartyOverride{Email: new("billing@example.org")},
		Tax:               &v1alpha1.InvoiceTaxOverride{},
		MonetaryPrecision: new(int32(0)),
		AdditionalLines:   []v1alpha1.InvoiceAdditionalLine{},
	}
	if config.Kind != "InvoiceConfig" || !reflect.DeepEqual(config.Invoice, want) {
		t.Fatalf("unexpected invoice configuration: %+v", config)
	}

	for _, precision := range []string{"2.0000000000000001", "1e-1000", "2.0"} {
		t.Run(precision, func(t *testing.T) {
			input := invoiceConfigHeader + "invoice: {monetaryPrecision: " + precision + "}\n"
			config, err := DecodeVersionedInvoiceConfig(strings.NewReader(input))
			if config != nil || !errors.Is(err, ErrDecode) {
				t.Fatalf("expected a decode error for precision %s, got %+v, %v", precision, config, err)
			}
			if !strings.Contains(err.Error(), "invoice.monetaryPrecision: must be an integer") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDecodeVersionedInvoiceConfigList(t *testing.T) {
	configList, err := DecodeVersionedInvoiceConfigList(strings.NewReader(invoiceConfigListHeader + "items: []\n"))
	if err != nil {
		t.Fatal(err)
	}
	if configList.Kind != "InvoiceConfigList" || !reflect.DeepEqual(configList.Items, []v1alpha1.InvoiceConfig{}) {
		t.Fatalf("unexpected invoice configuration list: %+v", configList)
	}

	t.Run("item-type-metadata", func(t *testing.T) {
		const items = "items: [{kind: InvoiceConfig, apiVersion: fyt.queueb.org/v1alpha1, invoice: {}}, {invoice: {}}]\n"
		configList, err := DecodeVersionedInvoiceConfigList(strings.NewReader(invoiceConfigListHeader + items))
		if err != nil || len(configList.Items) != 2 {
			t.Fatalf("expected two invoice configurations, got %+v, %v", configList, err)
		}
		item := configList.Items[0]
		if item.Kind != "InvoiceConfig" || item.APIVersion != "fyt.queueb.org/v1alpha1" {
			t.Errorf("unexpected item type metadata: %+v", item.TypeMeta)
		}
	})

	for _, tc := range []struct {
		items string
		want  string
	}{
		{"[{}]", "items[0].invoice: must be specified"},
		{"[null]", "items[0]: null"},
		{"[{invoice: {customer: null}}]", "items[0].invoice.customer: null"},
		{"[false]", "cannot unmarshal bool"},
		{"[{invoice: {}}, {invoice: {monetaryPrecision: 1e-1000}}]", "items[1].invoice.monetaryPrecision: must be an integer"},
		{"[{invoice: {}, kind: TimeEntry}]", "items[0].kind"},
		{"[{invoice: {}, apiVersion: fyt.queueb.org/v99}]", "items[0].apiVersion"},
	} {
		input := invoiceConfigListHeader + "items: " + tc.items + "\n"
		configList, err := DecodeVersionedInvoiceConfigList(strings.NewReader(input))
		if configList != nil || !errors.Is(err, ErrDecode) {
			t.Errorf("expected a decode error for %s, got %+v, %v", tc.items, configList, err)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("wanted %q, got %+v, %v", tc.want, configList, err)
		}
	}

	for _, tc := range []struct {
		item string
		key  string
	}{
		{`{"invoice":{},"kind":"TimeEntry"}`, "kind"},
		{`{"invoice":{},"apiVersion":"fyt.queueb.org/v99"}`, "apiVersion"},
	} {
		t.Run("json-"+tc.key, func(t *testing.T) {
			const listPrefix = `{"apiVersion":"fyt.queueb.org/v1alpha1","kind":"InvoiceConfigList","items":[`
			configList, err := DecodeVersionedInvoiceConfigList(strings.NewReader(listPrefix + tc.item + "]}"))
			if configList != nil || !errors.Is(err, ErrDecode) {
				t.Fatalf("expected a decode error for JSON item, got %+v, %v", configList, err)
			}
			if path := "items[0]." + tc.key; !strings.Contains(err.Error(), path) {
				t.Errorf("expected error on %s, got %v", path, err)
			}
		})
	}
}

func TestDecodeInvoiceSettingsOverride(t *testing.T) {
	for _, input := range []string{
		`{"invoice":{"hourlyRate":"0","monetaryPrecision":0,"customer":{"options":{}},"additionalLines":[]}}`,
		"invoice: {hourlyRate: '0', monetaryPrecision: 0, customer: {options: {}}, additionalLines: []}\n",
	} {
		got, err := DecodeInvoiceSettingsOverride(strings.NewReader(input))
		want := &v1alpha1.InvoiceSettingsOverride{
			HourlyRate:        new("0"),
			MonetaryPrecision: new(int32(0)),
			Customer:          &v1alpha1.InvoicePartyOverride{Options: map[string]string{}},
			AdditionalLines:   []v1alpha1.InvoiceAdditionalLine{},
		}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("expected %+v, got %+v, %v", want, got, err)
		}
	}
	for _, tc := range []struct{ input, want string }{
		{`{}`, "invoice: must be specified"},
		{`{"invoice":{},"kind":"InvoiceConfig"}`, `unknown field "kind"`},
		{`{"invoice":{"customer":null}}`, "invoice.customer: null"},
		{`{"invoice":{"period":"2026-10"}}`, `unknown field "invoice.period"`},
		{`{"invoice":{},"invoice":{}}`, "duplicate field"},
		{"invoice: {monetaryPrecision: 2.0000000000000001}\n", "invoice.monetaryPrecision"},
		{"invoice: {}\n---\ninvoice: {}\n", "exactly one YAML document"},
	} {
		override, err := DecodeInvoiceSettingsOverride(strings.NewReader(tc.input))
		if override != nil || !errors.Is(err, ErrDecode) {
			t.Fatalf("expected decode error for %q, got %+v, %v", tc.input, override, err)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("wanted %q, got %v", tc.want, err)
		}
	}
}

func TestReadInvoiceDocument(t *testing.T) {
	data, mapping, err := readInvoiceDocument(strings.NewReader("invoice: {}\n"))
	if err != nil || string(data) != `{"invoice":{}}` {
		t.Fatalf("unexpected document %q, %v", data, err)
	}
	if !reflect.DeepEqual(mapping, map[string]any{"invoice": map[string]any{}}) {
		t.Errorf("unexpected mapping: %+v", mapping)
	}
	for _, input := range []string{"[", "[]"} {
		if _, _, err := readInvoiceDocument(strings.NewReader(input)); !errors.Is(err, ErrDecode) {
			t.Errorf("expected a decode error for %q, got %v", input, err)
		}
	}
}

func TestDecodeInvoiceDocument(t *testing.T) {
	config, err := decodeInvoiceDocument[*v1alpha1.InvoiceConfig](strings.NewReader(invoiceConfigHeader + "invoice: {}\n"))
	if err != nil || !reflect.DeepEqual(config.Invoice, v1alpha1.InvoiceSettingsOverride{}) {
		t.Fatalf("expected an empty invoice configuration, got %+v, %v", config, err)
	}
	for _, tc := range []struct {
		name     string
		document string
		want     string
	}{
		{"wrong-root", "[]", "cannot unmarshal array"},
		{"null-root", "null", "must be an object"},
		{"wrong-kind", invoiceConfigListHeader, "unexpected object"},
		{"unsupported-version", "apiVersion: fyt.queueb.org/v9\nkind: InvoiceConfig\ninvoice: {}\n", "fyt.queueb.org/v9"},
		{"missing-version", "kind: InvoiceConfig\ninvoice: {}\n", "apiVersion"},
		{"missing-kind", "apiVersion: fyt.queueb.org/v1alpha1\ninvoice: {}\n", "Kind"},
		{"missing-invoice", invoiceConfigHeader, "invoice: must be specified"},
		{"invoice-null", invoiceConfigHeader + "invoice: null\n", "invoice: null"},
		{"nested-null", invoiceConfigHeader + "invoice: {customer: {email: null}}\n", "invoice.customer.email: null"},
		{"array-item-null", invoiceConfigHeader + "invoice: {additionalLines: [null]}\n", "invoice.additionalLines[0]: null"},
		{"unknown-field", invoiceConfigHeader + "invoice: {customer: {emial: x}}\n", "unknown field"},
		{"forbidden-period", invoiceConfigHeader + "invoice: {period: 2026-10}\n", "unknown field"},
		{"wrong-type", invoiceConfigHeader + "invoice: {hourlyRate: 0}\n", "cannot unmarshal number"},
		{"yaml-duplicate", invoiceConfigHeader + "invoice: {currency: USD, currency: EUR}\n", "currency"},
		{"yaml-non-string-key", invoiceConfigHeader + "invoice: {supplier: {options: {1: x, '1': y}}}\n", "invoice.supplier.options: mapping key 1 must be a string"},
		{"json-duplicate", `{"apiVersion":"fyt.queueb.org/v1alpha1","kind":"InvoiceConfig","invoice":{"tax":{"type":"None","type":"Percentage"}}}`, "duplicate field"},
		{"multiple-yaml-documents", invoiceConfigHeader + "invoice: {}\n---\n" + invoiceConfigHeader + "invoice: {}\n", "exactly one YAML document"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, err := decodeInvoiceDocument[*v1alpha1.InvoiceConfig](strings.NewReader(tc.document))
			if config != nil || !errors.Is(err, ErrDecode) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q, got %+v, %v", tc.want, config, err)
			}
		})
	}
}

func TestReadInvoiceJSON(t *testing.T) {
	const input = "{\"invoice\":{}}"
	data, err := readInvoiceJSON(strings.NewReader(input))
	if err != nil || string(data) != input {
		t.Fatalf("expected original JSON, got %q, %v", data, err)
	}
	data, err = readInvoiceJSON(strings.NewReader("# invoice overrides\n---\ninvoice: {}\n"))
	if err != nil || string(data) != input {
		t.Fatalf("expected YAML to become %q, got %q, %v", input, data, err)
	}
	if _, err := readInvoiceJSON(iotest.ErrReader(io.ErrUnexpectedEOF)); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected original read error, got %v", err)
	}
	for _, input := range []string{"[", "invoice: {}\n---\n[", "invoice: {}\n---\ninvoice: {}\n"} {
		if _, err := readInvoiceJSON(strings.NewReader(input)); err == nil {
			t.Errorf("expected error for %q", input)
		}
	}
	if _, err := readInvoiceJSON(strings.NewReader("invoice: {currency: null}\n")); err == nil {
		t.Error("expected a null value to fail YAML decoding")
	}
	if _, err := readInvoiceJSON(strings.NewReader("invoice: {monetaryPrecision: 2.0}\n")); err == nil {
		t.Error("expected floating-point monetary precision to fail YAML decoding")
	}
	data, err = readInvoiceJSON(strings.NewReader("---\nnull\n"))
	if err != nil || string(data) != "null" {
		t.Fatalf("expected a YAML null document to become JSON null, got %q, %v", data, err)
	}
}

func TestValidateYAMLInvoicePrecisions(t *testing.T) {
	t.Run("valid-precisions", func(t *testing.T) {
		documentMapping := map[any]any{
			"invoice": map[any]any{"monetaryPrecision": 0},
			"items":   []any{map[any]any{"invoice": map[any]any{"monetaryPrecision": 2}}},
		}
		if err := validateYAMLInvoicePrecisions(documentMapping); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("invalid-singleton-precision", func(t *testing.T) {
		documentMapping := map[any]any{"invoice": map[any]any{"monetaryPrecision": 2.0}}
		err := validateYAMLInvoicePrecisions(documentMapping)
		if err == nil || err.Error() != "invoice.monetaryPrecision: must be an integer" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("invalid-list-item-precision", func(t *testing.T) {
		documentMapping := map[any]any{
			"items": []any{
				map[any]any{"invoice": map[any]any{}},
				map[any]any{"invoice": map[any]any{"monetaryPrecision": 2.0}},
			},
		}
		err := validateYAMLInvoicePrecisions(documentMapping)
		if err == nil || err.Error() != "items[1].invoice.monetaryPrecision: must be an integer" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateYAMLInvoicePrecision(t *testing.T) {
	const path = "invoice.monetaryPrecision"
	for _, precision := range []int{0, 2} {
		settingsMapping := map[any]any{"monetaryPrecision": precision}
		if err := validateYAMLInvoicePrecision(settingsMapping, path); err != nil {
			t.Fatal(err)
		}
	}
	settingsMapping := map[any]any{"monetaryPrecision": 2.0}
	err := validateYAMLInvoicePrecision(settingsMapping, path)
	if err == nil || err.Error() != path+": must be an integer" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateInvoiceDocument(t *testing.T) {
	for _, tc := range []struct {
		name     string
		document map[string]any
		want     string
	}{
		{"nil", nil, "must be an object"},
		{"empty-invoice", map[string]any{"kind": "InvoiceConfig", "invoice": map[string]any{}}, ""},
		{"missing-invoice", map[string]any{"kind": "InvoiceConfig"}, "invoice: must be specified"},
		{"null-invoice", map[string]any{"kind": "InvoiceConfig", "invoice": nil}, "invoice: null"},
		{"valid-list", map[string]any{"kind": "InvoiceConfigList", "items": []any{map[string]any{"invoice": map[string]any{}}}}, ""},
		{"item-missing-invoice", map[string]any{"kind": "InvoiceConfigList", "items": []any{map[string]any{}}}, "items[0].invoice: must be specified"},
		{"item-null-invoice", map[string]any{"kind": "InvoiceConfigList", "items": []any{map[string]any{"invoice": nil}}}, "items[0].invoice: null"},
		{"item-wrong-type", map[string]any{"kind": "InvoiceConfigList", "items": []any{"invalid"}}, ""},
		{
			name: "item-wrong-kind",
			document: map[string]any{
				"kind":  "InvoiceConfigList",
				"items": []any{map[string]any{"invoice": map[string]any{}, "kind": "TimeEntry"}},
			},
			want: "items[0].kind: must be InvoiceConfig",
		},
		{
			name: "item-wrong-version",
			document: map[string]any{
				"kind":  "InvoiceConfigList",
				"items": []any{map[string]any{"invoice": map[string]any{}, "apiVersion": "fyt.queueb.org/v99"}},
			},
			want: "items[0].apiVersion: must be fyt.queueb.org/v1alpha1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInvoiceDocument(tc.document)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q, got %v", tc.want, err)
			}
		})
	}
}

func TestValidateInvoiceValues(t *testing.T) {
	document := map[string]any{
		"invoice": map[any]any{
			"options": map[string]any{},
			"lines":   []any{map[string]any{"value": ""}, map[string]any{"value": 0}},
		},
	}
	if err := validateInvoiceValues(document, ""); err != nil {
		t.Fatal(err)
	}
	invoice := document["invoice"].(map[any]any)
	invoice[1] = "numeric key"
	if err := validateInvoiceValues(document, ""); err == nil || err.Error() != "invoice: mapping key 1 must be a string" {
		t.Fatalf("unexpected error: %v", err)
	}
	delete(invoice, 1)
	invoice["lines"].([]any)[1] = nil
	if err := validateInvoiceValues(document, ""); err == nil || err.Error() != "invoice.lines[1]: null is not allowed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
