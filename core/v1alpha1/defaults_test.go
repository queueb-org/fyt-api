package v1alpha1

import (
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// WithNewLocalScheme creates local [runtime.Scheme] and installs
// there current package types.
func WithNewLocalScheme(t testing.TB) *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := Install(scheme); err != nil {
		t.Fatalf("could not install scheme: %v", err)
	}
	return scheme
}

func TestSetDefaults_Config(t *testing.T) {
	scheme := WithNewLocalScheme(t)
	cfg := &Config{}
	scheme.Default(cfg)
	expected := &Config{
		Documents: DocumentsConfig{OutputDirectory: "./documents/{{ .YearMonth }}"},
		Reports:   ReportsConfig{MemberName: new("John Doe")},
		Invoice: InvoiceSettings{
			Supplier:           InvoiceParty{Name: "John Doe"},
			Customer:           &InvoiceParty{Name: "ACME Inc"},
			ServiceDescription: "Software development",
			Currency:           new("USD"),
			HourlyRate:         "25",
			MonetaryPrecision:  new(int32(2)),
			Tax:                InvoiceTax{Type: TaxTypeNone},
		},
		Processing: ProcessingConfig{
			DuplicatePolicy:     DuplicatePolicyMerge,
			ShortEntryThreshold: &metav1.Duration{Duration: 5 * time.Minute},
		},
	}
	if !reflect.DeepEqual(cfg, expected) {
		t.Fatalf("want configuration %#v, got %#v", expected, cfg)
	}
	scheme.Default(cfg)
	if !reflect.DeepEqual(cfg, expected) {
		t.Fatal("reapplying defaults changed the configuration")
	}
}

func TestSetConfigProcessing(t *testing.T) {
	for _, entry := range []struct {
		name     string
		in       ProcessingConfig
		expected ProcessingConfig
	}{
		{
			name: "omitted",
			expected: ProcessingConfig{
				DuplicatePolicy:     DuplicatePolicyMerge,
				ShortEntryThreshold: &metav1.Duration{Duration: 5 * time.Minute},
			},
		},
		{
			name: "explicit-zero",
			in:   ProcessingConfig{ShortEntryThreshold: &metav1.Duration{}},
			expected: ProcessingConfig{
				DuplicatePolicy:     DuplicatePolicyMerge,
				ShortEntryThreshold: &metav1.Duration{},
			},
		},
		{
			name: "explicit-policy",
			in:   ProcessingConfig{DuplicatePolicy: DuplicatePolicyError},
			expected: ProcessingConfig{
				DuplicatePolicy:     DuplicatePolicyError,
				ShortEntryThreshold: &metav1.Duration{Duration: 5 * time.Minute},
			},
		},
		{
			name: "explicit-threshold",
			in:   ProcessingConfig{ShortEntryThreshold: &metav1.Duration{Duration: time.Minute}},
			expected: ProcessingConfig{
				DuplicatePolicy:     DuplicatePolicyMerge,
				ShortEntryThreshold: &metav1.Duration{Duration: time.Minute},
			},
		},
	} {
		t.Run(entry.name, func(t *testing.T) {
			cfg := &Config{Processing: entry.in}
			setConfigProcessing(cfg)
			if !reflect.DeepEqual(cfg.Processing, entry.expected) {
				t.Fatalf("want processing %#v, got %#v", entry.expected, cfg.Processing)
			}
			before := cfg.DeepCopy()
			setConfigProcessing(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}
}

func TestSetConfigReports(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       *string
		expected string
	}{
		{
			name:     "omitted",
			expected: "John Doe",
		},
		{
			name:     "explicit-empty",
			in:       new(""),
			expected: "",
		},
		{
			name:     "explicit-name",
			in:       new("Alex Example"),
			expected: "Alex Example",
		},
		{
			name:     "explicit-whitespace",
			in:       new(" \t\n"),
			expected: " \t\n",
		},
		{
			name:     "padded-name",
			in:       new(" Alex Example "),
			expected: " Alex Example ",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{Reports: ReportsConfig{MemberName: test.in}}
			setConfigReports(cfg)
			name := cfg.Reports.MemberName
			if name == nil {
				t.Fatalf("want member name %q, got nil", test.expected)
			}
			if *name != test.expected {
				t.Fatalf("want member name %q, got %q", test.expected, *name)
			}
			before := cfg.DeepCopy()
			setConfigReports(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}
}

func TestSetConfigInvoicePartyNames(t *testing.T) {
	for _, test := range []struct {
		name             string
		supplier         string
		customer         *InvoiceParty
		expectedSupplier string
	}{
		{
			name:             "omitted",
			expectedSupplier: "John Doe",
		},
		{
			name:             "explicit-supplier",
			supplier:         "Alex Example",
			expectedSupplier: "Alex Example",
		},
		{
			name:             "explicit-customer",
			customer:         &InvoiceParty{Name: "Example Company", Details: "Customer details"},
			expectedSupplier: "John Doe",
		},
		{
			name:             "explicit-names",
			supplier:         "Alex Example",
			customer:         &InvoiceParty{Name: "Example Company"},
			expectedSupplier: "Alex Example",
		},
		{
			name:             "explicit-empty-customer",
			customer:         &InvoiceParty{},
			expectedSupplier: "John Doe",
		},
		{
			name:             "explicit-customer-without-name",
			customer:         &InvoiceParty{Details: "Customer details"},
			expectedSupplier: "John Doe",
		},
		{
			name:             "explicit-whitespace",
			supplier:         " \t\n",
			customer:         &InvoiceParty{Name: " \r\n\u00a0\u2003"},
			expectedSupplier: " \t\n",
		},
		{
			name:             "padded-names",
			supplier:         " Alex Example ",
			customer:         &InvoiceParty{Name: " Example Company "},
			expectedSupplier: " Alex Example ",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{
				Reports: ReportsConfig{MemberName: new("Report Member")},
				Invoice: InvoiceSettings{
					Supplier: InvoiceParty{Name: test.supplier, Details: "Supplier details"},
					Customer: test.customer,
				},
			}
			expectedCustomer := test.customer.DeepCopy()
			if expectedCustomer == nil {
				expectedCustomer = &InvoiceParty{Name: "ACME Inc"}
			}
			setConfigInvoice(cfg)
			expected := InvoiceSettings{
				Supplier:           InvoiceParty{Name: test.expectedSupplier, Details: "Supplier details"},
				Customer:           expectedCustomer,
				ServiceDescription: "Software development",
				Currency:           new("USD"),
				HourlyRate:         "25",
				MonetaryPrecision:  new(int32(2)),
				Tax:                InvoiceTax{Type: TaxTypeNone},
			}
			if !reflect.DeepEqual(cfg.Invoice, expected) {
				t.Fatalf("want invoice %#v, got %#v", expected, cfg.Invoice)
			}
			if name := cfg.Reports.MemberName; name == nil || *name != "Report Member" {
				t.Fatal("invoice defaults changed the report member name")
			}
			before := cfg.DeepCopy()
			setConfigInvoice(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}
}

func TestSetConfigInvoiceServiceDescription(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "omitted",
			expected: "Software development",
		},
		{
			name:     "explicit-description",
			in:       "Technical consulting",
			expected: "Technical consulting",
		},
		{
			name:     "explicit-whitespace",
			in:       " \t\n\u00a0\u2003",
			expected: " \t\n\u00a0\u2003",
		},
		{
			name:     "padded-description",
			in:       " Technical consulting ",
			expected: " Technical consulting ",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{Invoice: InvoiceSettings{ServiceDescription: test.in}}
			setConfigInvoice(cfg)
			if cfg.Invoice.ServiceDescription != test.expected {
				t.Fatalf("want service description %q, got %q", test.expected, cfg.Invoice.ServiceDescription)
			}
			before := cfg.DeepCopy()
			setConfigInvoice(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}
}

func TestSetConfigInvoiceCurrency(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       *string
		expected string
	}{
		{
			name:     "omitted",
			expected: "USD",
		},
		{
			name:     "explicit-currency",
			in:       new("EUR"),
			expected: "EUR",
		},
		{
			name:     "lowercase-currency",
			in:       new("usd"),
			expected: "usd",
		},
		{
			name: "explicit-empty",
			in:   new(""),
		},
		{
			name:     "explicit-whitespace",
			in:       new(" \t\n\u00a0\u2003"),
			expected: " \t\n\u00a0\u2003",
		},
		{
			name:     "padded-currency",
			in:       new(" EUR "),
			expected: " EUR ",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{Invoice: InvoiceSettings{Currency: test.in}}
			setConfigInvoice(cfg)
			if cfg.Invoice.Currency == nil || *cfg.Invoice.Currency != test.expected {
				t.Fatalf("want currency %q, got %#v", test.expected, cfg.Invoice.Currency)
			}
			before := cfg.DeepCopy()
			setConfigInvoice(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}
}

func TestSetConfigInvoiceHourlyRate(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "omitted",
			expected: "25",
		},
		{
			name:     "explicit-zero",
			in:       "0",
			expected: "0",
		},
		{
			name:     "explicit-decimal",
			in:       "50.000",
			expected: "50.000",
		},
		{
			name:     "explicit-negative",
			in:       "-1",
			expected: "-1",
		},
		{
			name:     "explicit-whitespace",
			in:       " \t\n",
			expected: " \t\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{Invoice: InvoiceSettings{HourlyRate: test.in}}
			setConfigInvoice(cfg)
			if cfg.Invoice.HourlyRate != test.expected {
				t.Fatalf("want hourly rate %q, got %q", test.expected, cfg.Invoice.HourlyRate)
			}
			before := cfg.DeepCopy()
			setConfigInvoice(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}
}

func TestSetConfigInvoiceMonetaryPrecision(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       *int32
		expected int32
	}{
		{
			name:     "omitted",
			expected: 2,
		},
		{
			name:     "explicit-zero",
			in:       new(int32(0)),
			expected: 0,
		},
		{
			name:     "explicit-maximum",
			in:       new(int32(6)),
			expected: 6,
		},
		{
			name:     "explicit-negative",
			in:       new(int32(-1)),
			expected: -1,
		},
		{
			name:     "explicit-above-maximum",
			in:       new(int32(7)),
			expected: 7,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{Invoice: InvoiceSettings{MonetaryPrecision: test.in}}
			setConfigInvoice(cfg)
			if cfg.Invoice.MonetaryPrecision == nil || *cfg.Invoice.MonetaryPrecision != test.expected {
				t.Fatalf("want monetary precision %d, got %v", test.expected, cfg.Invoice.MonetaryPrecision)
			}
			if test.in != nil && cfg.Invoice.MonetaryPrecision != test.in {
				t.Fatal("defaults replaced an explicit monetary precision")
			}
			before := cfg.DeepCopy()
			setConfigInvoice(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}

	t.Run("independent-defaults", func(t *testing.T) {
		first, second := &Config{}, &Config{}
		setConfigInvoice(first)
		setConfigInvoice(second)
		*first.Invoice.MonetaryPrecision = 0
		if *second.Invoice.MonetaryPrecision != 2 {
			t.Fatal("changing one configuration affected another configuration's defaults")
		}
	})
}

func TestSetConfigInvoiceTaxType(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       TaxType
		expected TaxType
	}{
		{
			name:     "omitted",
			expected: TaxTypeNone,
		},
		{
			name:     "explicit-none",
			in:       TaxTypeNone,
			expected: TaxTypeNone,
		},
		{
			name:     "explicit-percentage",
			in:       TaxTypePercentage,
			expected: TaxTypePercentage,
		},
		{
			name:     "explicit-unknown",
			in:       "Other",
			expected: "Other",
		},
		{
			name:     "explicit-whitespace",
			in:       " \t\n",
			expected: " \t\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{Invoice: InvoiceSettings{Tax: InvoiceTax{Type: test.in, RatePercent: "7.500"}}}
			setConfigInvoice(cfg)
			if cfg.Invoice.Tax.Type != test.expected {
				t.Fatalf("want tax type %q, got %q", test.expected, cfg.Invoice.Tax.Type)
			}
			if cfg.Invoice.Tax.RatePercent != "7.500" {
				t.Fatal("tax type default changed the tax percentage")
			}
			before := cfg.DeepCopy()
			setConfigInvoice(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}
}

func TestSetConfigInvoiceTaxRatePercent(t *testing.T) {
	for _, test := range []struct {
		name string
		rate string
	}{
		{name: "omitted"},
		{name: "explicit-zero", rate: "0"},
		{name: "explicit-decimal", rate: "7.500"},
		{name: "explicit-negative", rate: "-1"},
		{name: "explicit-whitespace", rate: " \t\n"},
	} {
		for _, taxType := range []TaxType{TaxTypeNone, TaxTypePercentage} {
			t.Run(test.name+"/"+string(taxType), func(t *testing.T) {
				cfg := &Config{Invoice: InvoiceSettings{Tax: InvoiceTax{Type: taxType, RatePercent: test.rate}}}
				setConfigInvoice(cfg)
				if cfg.Invoice.Tax.RatePercent != test.rate {
					t.Fatalf("want tax percentage %q, got %q", test.rate, cfg.Invoice.Tax.RatePercent)
				}
				if cfg.Invoice.Tax.Type != taxType {
					t.Fatal("tax percentage default changed the tax type")
				}
				before := cfg.DeepCopy()
				setConfigInvoice(cfg)
				if !reflect.DeepEqual(cfg, before) {
					t.Fatal("reapplying defaults changed the configuration")
				}
			})
		}
	}
}

func TestSetConfigDocuments(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "omitted",
			expected: "./documents/{{ .YearMonth }}",
		},
		{
			name:     "relative-path",
			in:       "../invoices",
			expected: "../invoices",
		},
		{
			name:     "path-template",
			in:       "../invoices/{{ .Year }}/{{ .Month }}",
			expected: "../invoices/{{ .Year }}/{{ .Month }}",
		},
		{
			name:     "absolute-path",
			in:       "/tmp/documents",
			expected: "/tmp/documents",
		},
		{
			name:     "padded-path",
			in:       " ./documents ",
			expected: " ./documents ",
		},
		{
			name:     "explicit-whitespace",
			in:       " \t\n\u00a0\u2003",
			expected: " \t\n\u00a0\u2003",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{Documents: DocumentsConfig{
				OutputDirectory: test.in,
				TemplatesDirectories: []Directory{
					{Path: "./templates", Description: "Custom templates"},
					{Path: "./shared/templates", Description: "Shared templates"},
				},
			}}
			setConfigDocuments(cfg)
			expected := DocumentsConfig{
				OutputDirectory: test.expected,
				TemplatesDirectories: []Directory{
					{Path: "./templates", Description: "Custom templates"},
					{Path: "./shared/templates", Description: "Shared templates"},
				},
			}
			if !reflect.DeepEqual(cfg.Documents, expected) {
				t.Fatalf("want documents %#v, got %#v", expected, cfg.Documents)
			}
			before := cfg.DeepCopy()
			setConfigDocuments(cfg)
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("reapplying defaults changed the configuration")
			}
		})
	}
}

func TestSetDefaults_TimeEntry(t *testing.T) {
	scheme := WithNewLocalScheme(t)

	for _, entry := range []struct {
		name     string
		in       *TimeEntry
		expected *TimeEntry
	}{
		{
			name:     "defaults",
			in:       &TimeEntry{},
			expected: &TimeEntry{},
		},
	} {
		t.Run(entry.name, func(in *testing.T) {
			// runs default operation.
			scheme.Default(entry.in)

			if !reflect.DeepEqual(entry.in, entry.expected) {
				in.Errorf("\nexp:\n%#v\ngot:\n%#v\n", entry.expected, entry.in)
			}
		})
	}
}
