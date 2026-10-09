package validation

import (
	"reflect"
	"testing"
	"time"

	"fyt.queueb.org/api/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// test

func TestOnCreateConfig(t *testing.T) {
	t.Run("ok", func(in *testing.T) {
		errs := OnCreateConfig(withValidConfig())
		if len(errs) != 0 {
			in.Error("expected zero errors on a valid resource")
		}
	})

	t.Run("errors-from-multiple-sections", func(t *testing.T) {
		cfg := withValidConfig()
		cfg.Processing.DuplicatePolicy = "Ignore"
		cfg.InputDirectories = []core.Directory{
			{Path: " "},
		}
		cfg.Reports.MemberName = ""
		cfg.Invoice.Currency = new("")
		cfg.Documents.OutputDirectory = ""
		expected := []string{
			"processing.duplicatePolicy",
			"inputDirectories[0].path",
			"reports.memberName",
			"invoice.currency",
			"documents.outputDirectory",
		}
		errs := OnCreateConfig(cfg)
		if len(errs) != len(expected) {
			t.Fatalf("want errors for fields %v, got %#v", expected, errs)
		}
		for i, path := range expected {
			if errs[i].Field != path {
				t.Errorf("want error for field %s, got %#v", path, errs[i])
			}
		}
	})
}

func TestValidateConfig(t *testing.T) {
	cfg := withValidConfig()
	cfg.Processing.DuplicatePolicy = "Ignore"
	cfg.Reports.MemberName = " "
	cfg.Invoice.HourlyRate = "-1"
	cfg.Documents.OutputDirectory = ""

	for _, test := range []struct {
		name     string
		options  ConfigValidationOptions
		expected []string
	}{
		{"shared", ConfigValidationOptions{}, []string{"processing.duplicatePolicy", "documents.outputDirectory"}},
		{"reports", ConfigValidationOptions{Reports: true}, []string{"processing.duplicatePolicy", "reports.memberName", "documents.outputDirectory"}},
		{"invoice", ConfigValidationOptions{Invoice: true}, []string{"processing.duplicatePolicy", "invoice.hourlyRate", "documents.outputDirectory"}},
		{"all", ConfigValidationOptions{Reports: true, Invoice: true}, []string{"processing.duplicatePolicy", "reports.memberName", "invoice.hourlyRate", "documents.outputDirectory"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			errs := ValidateConfig(cfg, test.options)
			if len(errs) != len(test.expected) {
				t.Fatalf("want errors for fields %v, got %v", test.expected, errs)
			}
			for i, path := range test.expected {
				if errs[i].Field != path {
					t.Errorf("want error for field %s, got %v", path, errs[i])
				}
			}
		})
	}
}

func TestValidateConfigReports(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       core.ReportsConfig
		expected []string
	}{
		{
			name: "valid",
			in:   core.ReportsConfig{MemberName: "John Doe"},
		},
		{
			name: "invalid-member-name",
			in:   core.ReportsConfig{MemberName: " "},
			expected: []string{
				"reports.memberName",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &core.Config{Reports: test.in}
			before := cfg.DeepCopy()
			errs := validateConfigReports(cfg)
			if len(errs) != len(test.expected) {
				t.Fatalf("want errors for fields %v, got %#v", test.expected, errs)
			}
			for i, path := range test.expected {
				if errs[i].Field != path {
					t.Errorf("want error for field %s, got %#v", path, errs[i])
				}
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed reports settings")
			}
		})
	}
}

func TestValidateConfigProcessing(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       core.ProcessingConfig
		expected []string
	}{
		{
			name: "valid-zero-threshold",
			in: core.ProcessingConfig{
				DuplicatePolicy:     core.DuplicatePolicyError,
				ShortEntryThreshold: &metav1.Duration{},
			},
		},
		{
			name: "both-invalid",
			in: core.ProcessingConfig{
				DuplicatePolicy:     "Ignore",
				ShortEntryThreshold: &metav1.Duration{Duration: -time.Minute},
			},
			expected: []string{
				"processing.duplicatePolicy",
				"processing.shortEntryThreshold",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &core.Config{Processing: test.in}
			before := cfg.DeepCopy()
			errs := validateConfigProcessing(cfg)
			if len(errs) != len(test.expected) {
				t.Fatalf("want errors for fields %v, got %#v", test.expected, errs)
			}
			for i, path := range test.expected {
				if errs[i].Field != path {
					t.Errorf("want error for field %s, got %#v", path, errs[i])
				}
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed processing settings")
			}
		})
	}
}

func TestValidateInvoiceConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       core.InvoiceSettings
		expected []string
	}{
		{
			name: "valid",
			in:   withValidConfig().Invoice,
		},
		{
			name: "invalid-fields",
			in: core.InvoiceSettings{
				Supplier:           core.InvoiceParty{Name: " "},
				Customer:           &core.InvoiceParty{},
				ServiceDescription: "\t",
				Currency:           new(" "),
				HourlyRate:         "-1",
				MonetaryPrecision:  new(int32(7)),
				Tax:                core.InvoiceTax{Type: "Other", RatePercent: "-1"},
			},
			expected: []string{
				"invoice.supplier.name",
				"invoice.customer.name",
				"invoice.serviceDescription",
				"invoice.currency",
				"invoice.hourlyRate",
				"invoice.monetaryPrecision",
				"invoice.tax.type",
				"invoice.tax.ratePercent",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &core.Config{Invoice: test.in}
			before := cfg.DeepCopy()
			errs := ValidateInvoiceConfig(cfg)
			if len(errs) != len(test.expected) {
				t.Fatalf("want errors for fields %v, got %#v", test.expected, errs)
			}
			for i, path := range test.expected {
				if errs[i].Field != path {
					t.Errorf("want error for field %s, got %#v", path, errs[i])
				}
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed invoice settings")
			}
		})
	}
}

func TestValidateConfigAdditionalInvoiceLines(t *testing.T) {
	cfg := withValidConfig()
	cfg.Invoice.AdditionalLines = []core.InvoiceAdditionalLine{
		{Description: "Free license", Quantity: "2", UnitPrice: "0"},
		{Description: "Support", Quantity: "0.25", UnitPrice: "10.00"},
		{Description: " ", Quantity: "000.00", UnitPrice: "-1"},
		{Description: "Storage", Quantity: "1/2", UnitPrice: ""},
	}
	expected := []struct {
		field string
		kind  field.ErrorType
	}{
		{"invoice.additionalLines[2].description", field.ErrorTypeRequired},
		{"invoice.additionalLines[2].quantity", field.ErrorTypeInvalid},
		{"invoice.additionalLines[2].unitPrice", field.ErrorTypeInvalid},
		{"invoice.additionalLines[3].quantity", field.ErrorTypeInvalid},
		{"invoice.additionalLines[3].unitPrice", field.ErrorTypeInvalid},
	}
	errs := validateConfigAdditionalInvoiceLines(cfg)
	if len(errs) != len(expected) {
		t.Fatalf("expected %d errors, got %v", len(expected), errs)
	}
	for i, want := range expected {
		if errs[i].Field != want.field || errs[i].Type != want.kind {
			t.Errorf("expected %s for %s, got %#v", want.kind, want.field, errs[i])
		}
	}
}

func TestValidateConfigDocuments(t *testing.T) {
	for _, test := range []struct {
		name     string
		in       core.DocumentsConfig
		expected []string
	}{
		{
			name: "valid-empty-templates",
			in: core.DocumentsConfig{
				OutputDirectory:      "./documents",
				TemplatesDirectories: []core.Directory{},
			},
		},
		{
			name: "both-invalid",
			in: core.DocumentsConfig{
				OutputDirectory: " ",
				TemplatesDirectories: []core.Directory{
					{Path: "./templates"},
					{Path: " "},
				},
			},
			expected: []string{
				"documents.outputDirectory",
				"documents.templatesDirectories[1].path",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &core.Config{Documents: test.in}
			before := cfg.DeepCopy()
			errs := validateConfigDocuments(cfg)
			if len(errs) != len(test.expected) {
				t.Fatalf("want errors for fields %v, got %#v", test.expected, errs)
			}
			for i, path := range test.expected {
				if errs[i].Field != path {
					t.Errorf("want error for field %s, got %#v", path, errs[i])
				}
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed documents settings")
			}
		})
	}
}

func TestValidateConfigOutputDirectory(t *testing.T) {
	for _, test := range []struct {
		name      string
		directory string
		expected  field.ErrorList
	}{
		{
			name:      "relative-path",
			directory: "./documents",
		},
		{
			name:      "absolute-path",
			directory: "/tmp/documents",
		},
		{
			name:      "path-with-spaces",
			directory: "./my documents",
		},
		{
			name:      "padded-path",
			directory: " ./documents ",
		},
		{
			name: "empty-path",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("documents", "outputDirectory"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:      "spaces-only",
			directory: "   ",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("documents", "outputDirectory"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:      "whitespace-only",
			directory: " \t\r\n\u00a0\u2003",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("documents", "outputDirectory"),
					"must not be empty or contain only whitespace",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Documents.OutputDirectory = test.directory
			before := cfg.DeepCopy()
			errs := validateConfigOutputDirectory(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed document settings")
			}
		})
	}
}

func TestValidateConfigTaxRatePercent(t *testing.T) {
	for _, test := range []struct {
		name  string
		rate  string
		valid bool
	}{
		{name: "zero", rate: "0", valid: true},
		{name: "decimal-zero", rate: "0.000", valid: true},
		{name: "integer", rate: "5", valid: true},
		{name: "decimal", rate: "7.500", valid: true},
		{name: "above-one-hundred", rate: "125.50", valid: true},
		{name: "leading-zeroes", rate: "007.500", valid: true},
		{name: "small-exact-percentage", rate: "0.00000000000000000000000001", valid: true},
		{name: "large-exact-percentage", rate: "123456789012345678901234567890.123456789000", valid: true},
		{name: "empty"},
		{name: "whitespace", rate: " \t\r\n\u00a0"},
		{name: "negative", rate: "-7.50"},
		{name: "positive-sign", rate: "+5"},
		{name: "padded", rate: " 5 "},
		{name: "trailing-newline", rate: "5\n"},
		{name: "missing-integer", rate: ".50"},
		{name: "missing-fraction", rate: "5."},
		{name: "comma", rate: "7,50"},
		{name: "percent-symbol", rate: "7.5%"},
		{name: "exponent", rate: "5e1"},
		{name: "fraction", rate: "1/2"},
		{name: "hexadecimal", rate: "0x05"},
		{name: "nan", rate: "NaN"},
		{name: "infinity", rate: "Inf"},
		{name: "negative-infinity", rate: "-Inf"},
		{name: "text", rate: "invalid"},
	} {
		for _, taxType := range []core.TaxType{core.TaxTypeNone, core.TaxTypePercentage} {
			t.Run(test.name+"/"+string(taxType), func(t *testing.T) {
				cfg := withValidConfig()
				cfg.Invoice.Tax = core.InvoiceTax{Type: taxType, RatePercent: test.rate}
				before := cfg.DeepCopy()
				errs := validateConfigTaxRatePercent(cfg)
				expected := field.ErrorList{}
				if taxType == core.TaxTypePercentage && !test.valid {
					expected = field.ErrorList{
						field.Invalid(
							field.NewPath("invoice", "tax", "ratePercent"), test.rate,
							"must be a non-negative decimal number, such as 0, 5 or 7.50",
						),
					}
				}
				if !reflect.DeepEqual(errs, expected) {
					t.Fatalf("want errors %#v, got %#v", expected, errs)
				}
				if !reflect.DeepEqual(cfg, before) {
					t.Fatal("validation changed tax settings")
				}
			})
		}
	}
}

func TestValidateConfigTaxType(t *testing.T) {
	for _, test := range []struct {
		name     string
		taxType  core.TaxType
		expected field.ErrorList
	}{
		{
			name:    "none",
			taxType: core.TaxTypeNone,
		},
		{
			name:    "percentage",
			taxType: core.TaxTypePercentage,
		},
		{
			name:    "unknown-type",
			taxType: "Other",
			expected: field.ErrorList{
				field.NotSupported(
					field.NewPath("invoice", "tax", "type"), core.TaxType("Other"),
					[]core.TaxType{core.TaxTypeNone, core.TaxTypePercentage},
				),
			},
		},
		{
			name:    "lowercase-type",
			taxType: "none",
			expected: field.ErrorList{
				field.NotSupported(
					field.NewPath("invoice", "tax", "type"), core.TaxType("none"),
					[]core.TaxType{core.TaxTypeNone, core.TaxTypePercentage},
				),
			},
		},
		{
			name: "empty-type",
			expected: field.ErrorList{
				field.NotSupported(
					field.NewPath("invoice", "tax", "type"), core.TaxType(""),
					[]core.TaxType{core.TaxTypeNone, core.TaxTypePercentage},
				),
			},
		},
		{
			name:    "whitespace-type",
			taxType: " \t\n",
			expected: field.ErrorList{
				field.NotSupported(
					field.NewPath("invoice", "tax", "type"), core.TaxType(" \t\n"),
					[]core.TaxType{core.TaxTypeNone, core.TaxTypePercentage},
				),
			},
		},
		{
			name:    "padded-type",
			taxType: " None ",
			expected: field.ErrorList{
				field.NotSupported(
					field.NewPath("invoice", "tax", "type"), core.TaxType(" None "),
					[]core.TaxType{core.TaxTypeNone, core.TaxTypePercentage},
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Invoice.Tax.Type = test.taxType
			before := cfg.DeepCopy()
			errs := validateConfigTaxType(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed tax settings")
			}
		})
	}
}

func TestValidateConfigMonetaryPrecision(t *testing.T) {
	for _, test := range []struct {
		name      string
		precision *int32
		expected  field.ErrorList
	}{
		{
			name: "unspecified",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "monetaryPrecision"),
					"must be specified",
				),
			},
		},
		{
			name:      "minimum",
			precision: new(int32(0)),
		},
		{
			name:      "default",
			precision: new(int32(2)),
		},
		{
			name:      "maximum",
			precision: new(int32(6)),
		},
		{
			name:      "below-minimum",
			precision: new(int32(-1)),
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("invoice", "monetaryPrecision"), int32(-1),
					"must be between 0 and 6, inclusive",
				),
			},
		},
		{
			name:      "above-maximum",
			precision: new(int32(7)),
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("invoice", "monetaryPrecision"), int32(7),
					"must be between 0 and 6, inclusive",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Invoice.MonetaryPrecision = test.precision
			before := cfg.DeepCopy()
			errs := validateConfigMonetaryPrecision(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed the monetary precision")
			}
		})
	}
}

func TestValidateConfigHourlyRate(t *testing.T) {
	for _, test := range []struct {
		name  string
		rate  string
		valid bool
	}{
		{name: "integer", rate: "25", valid: true},
		{name: "zero", rate: "0", valid: true},
		{name: "decimal-zero", rate: "0.00", valid: true},
		{name: "decimal", rate: "25.50", valid: true},
		{name: "leading-zeroes", rate: "001.2500", valid: true},
		{name: "small-exact-rate", rate: "0.00000000000000000000000001", valid: true},
		{name: "large-exact-rate", rate: "123456789012345678901234567890.123456789000", valid: true},
		{name: "empty"},
		{name: "whitespace", rate: " \t\r\n\u00a0"},
		{name: "negative", rate: "-25.50"},
		{name: "positive-sign", rate: "+25"},
		{name: "padded", rate: " 25 "},
		{name: "trailing-newline", rate: "25\n"},
		{name: "missing-integer", rate: ".50"},
		{name: "missing-fraction", rate: "25."},
		{name: "comma", rate: "25,50"},
		{name: "exponent", rate: "2.5e1"},
		{name: "fraction", rate: "1/2"},
		{name: "hexadecimal", rate: "0x19"},
		{name: "nan", rate: "NaN"},
		{name: "infinity", rate: "Inf"},
		{name: "negative-infinity", rate: "-Inf"},
		{name: "text", rate: "invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Invoice.HourlyRate = test.rate
			errs := validateConfigHourlyRate(cfg)
			expected := field.ErrorList{}
			if !test.valid {
				expected = field.ErrorList{
					field.Invalid(
						field.NewPath("invoice", "hourlyRate"), test.rate,
						"must be a non-negative decimal number, such as 0, 25 or 25.50",
					),
				}
			}
			if !reflect.DeepEqual(errs, expected) {
				t.Fatalf("want errors %#v, got %#v", expected, errs)
			}
			if cfg.Invoice.HourlyRate != test.rate {
				t.Fatal("validation changed the hourly rate")
			}
		})
	}
}

func TestValidateConfigCurrency(t *testing.T) {
	for _, test := range []struct {
		name     string
		currency *string
		expected field.ErrorList
	}{
		{
			name:     "valid-currency",
			currency: new("USD"),
		},
		{
			name:     "other-currency",
			currency: new("EUR"),
		},
		{
			name:     "lowercase-currency",
			currency: new("usd"),
		},
		{
			name:     "padded-currency",
			currency: new(" EUR "),
		},
		{
			name: "missing-currency",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "currency"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:     "empty-currency",
			currency: new(""),
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "currency"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:     "spaces-only",
			currency: new("   "),
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "currency"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:     "whitespace-only",
			currency: new(" \t\r\n\u00a0\u2003"),
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "currency"),
					"must not be empty or contain only whitespace",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Invoice.Currency = test.currency
			errs := validateConfigCurrency(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if cfg.Invoice.Currency != test.currency {
				t.Fatal("validation changed the currency")
			}
		})
	}
}

func TestValidateConfigServiceDescription(t *testing.T) {
	for _, test := range []struct {
		name        string
		description string
		expected    field.ErrorList
	}{
		{
			name:        "valid-description",
			description: "Software development",
		},
		{
			name:        "padded-description",
			description: " Technical consulting ",
		},
		{
			name: "empty-description",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "serviceDescription"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:        "spaces-only",
			description: "   ",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "serviceDescription"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:        "whitespace-only",
			description: " \t\r\n\u00a0\u2003",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "serviceDescription"),
					"must not be empty or contain only whitespace",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Invoice.ServiceDescription = test.description
			errs := validateConfigServiceDescription(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if cfg.Invoice.ServiceDescription != test.description {
				t.Fatal("validation changed the service description")
			}
		})
	}
}

func TestValidateConfigSupplierName(t *testing.T) {
	for _, test := range []struct {
		name      string
		partyName string
		expected  field.ErrorList
	}{
		{
			name:      "valid-name",
			partyName: "John Doe",
		},
		{
			name:      "padded-name",
			partyName: " John Doe ",
		},
		{
			name: "empty-name",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "supplier", "name"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:      "spaces-only",
			partyName: "   ",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "supplier", "name"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:      "whitespace-only",
			partyName: " \t\r\n\u00a0\u2003",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("invoice", "supplier", "name"),
					"must not be empty or contain only whitespace",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Invoice.Supplier.Name = test.partyName
			before := cfg.DeepCopy()
			errs := validateConfigSupplierName(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed invoice settings")
			}
		})
	}
}

func TestValidateConfigCustomerName(t *testing.T) {
	requiredName := field.ErrorList{
		field.Required(field.NewPath("invoice", "customer", "name"), "must not be empty or contain only whitespace"),
	}
	for _, test := range []struct {
		name     string
		customer *core.InvoiceParty
		expected field.ErrorList
	}{
		{name: "valid-name", customer: &core.InvoiceParty{Name: "ACME Inc"}},
		{name: "padded-name", customer: &core.InvoiceParty{Name: " ACME Inc "}},
		{name: "missing-customer", expected: requiredName},
		{name: "empty-name", customer: &core.InvoiceParty{}, expected: requiredName},
		{name: "spaces-only", customer: &core.InvoiceParty{Name: "   "}, expected: requiredName},
		{name: "whitespace-only", customer: &core.InvoiceParty{Name: " \t\r\n\u00a0\u2003"}, expected: requiredName},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Invoice.Customer = test.customer
			before := cfg.DeepCopy()
			errs := validateConfigCustomerName(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed invoice settings")
			}
		})
	}
}

func TestValidateConfigMemberName(t *testing.T) {
	for _, test := range []struct {
		name       string
		memberName string
		expected   field.ErrorList
	}{
		{
			name:       "valid-name",
			memberName: "John Doe",
		},
		{
			name:       "unicode-name",
			memberName: "Иван Иванов",
		},
		{
			name:       "padded-name",
			memberName: " John Doe ",
		},
		{
			name: "empty-name",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("reports", "memberName"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:       "spaces-only",
			memberName: "   ",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("reports", "memberName"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name:       "whitespace-only",
			memberName: " \t\r\n\u00a0\u2003",
			expected: field.ErrorList{
				field.Required(
					field.NewPath("reports", "memberName"),
					"must not be empty or contain only whitespace",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Reports.MemberName = test.memberName
			errs := validateConfigMemberName(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if cfg.Reports.MemberName != test.memberName {
				t.Fatal("validation changed the member name")
			}
		})
	}
}

func TestValidateConfigTemplatesDirectories(t *testing.T) {
	for _, test := range []struct {
		name        string
		directories []core.Directory
		expected    field.ErrorList
	}{
		{
			name: "unspecified",
		},
		{
			name:        "empty-list",
			directories: []core.Directory{},
		},
		{
			name: "nonblank-paths",
			directories: []core.Directory{
				{Path: "./timesheets"},
				{Path: "/timesheets"},
				{Path: "./my timesheets"},
				{Path: " ./timesheets "},
			},
		},
		{
			name: "empty-path",
			directories: []core.Directory{
				{},
			},
			expected: field.ErrorList{
				field.Required(
					field.NewPath("documents", "templatesDirectories").Index(0).Child("path"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name: "whitespace-path",
			directories: []core.Directory{
				{Path: " \t\r\n\u00a0\u2003"},
			},
			expected: field.ErrorList{
				field.Required(
					field.NewPath("documents", "templatesDirectories").Index(0).Child("path"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name: "multiple-invalid-paths",
			directories: []core.Directory{
				{Path: ""},
				{Path: "./timesheets"},
				{Path: "   "},
			},
			expected: field.ErrorList{
				field.Required(
					field.NewPath("documents", "templatesDirectories").Index(0).Child("path"),
					"must not be empty or contain only whitespace",
				),
				field.Required(
					field.NewPath("documents", "templatesDirectories").Index(2).Child("path"),
					"must not be empty or contain only whitespace",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Documents.TemplatesDirectories = test.directories
			before := cfg.DeepCopy()
			errs := validateConfigTemplatesDirectories(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed template directory paths")
			}
		})
	}
}

func TestValidateConfigInputDirectories(t *testing.T) {
	for _, test := range []struct {
		name        string
		directories []core.Directory
		expected    field.ErrorList
	}{
		{
			name: "unspecified",
		},
		{
			name:        "empty-list",
			directories: []core.Directory{},
		},
		{
			name: "nonblank-paths",
			directories: []core.Directory{
				{Path: "./timesheets"},
				{Path: "/timesheets"},
				{Path: "./my timesheets"},
				{Path: " ./timesheets "},
			},
		},
		{
			name: "empty-path",
			directories: []core.Directory{
				{},
			},
			expected: field.ErrorList{
				field.Required(
					field.NewPath("inputDirectories").Index(0).Child("path"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name: "whitespace-path",
			directories: []core.Directory{
				{Path: " \t\r\n\u00a0\u2003"},
			},
			expected: field.ErrorList{
				field.Required(
					field.NewPath("inputDirectories").Index(0).Child("path"),
					"must not be empty or contain only whitespace",
				),
			},
		},
		{
			name: "multiple-invalid-paths",
			directories: []core.Directory{
				{Path: ""},
				{Path: "./timesheets"},
				{Path: "   "},
			},
			expected: field.ErrorList{
				field.Required(
					field.NewPath("inputDirectories").Index(0).Child("path"),
					"must not be empty or contain only whitespace",
				),
				field.Required(
					field.NewPath("inputDirectories").Index(2).Child("path"),
					"must not be empty or contain only whitespace",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.InputDirectories = test.directories
			before := cfg.DeepCopy()
			errs := validateConfigInputDirectories(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
			if !reflect.DeepEqual(cfg, before) {
				t.Fatal("validation changed input directory paths")
			}
		})
	}
}

func TestValidateConfigDuplicatePolicy(t *testing.T) {
	for _, test := range []struct {
		name     string
		policy   core.DuplicatePolicy
		expected field.ErrorList
	}{
		{
			name:   "merge",
			policy: core.DuplicatePolicyMerge,
		},
		{
			name:   "error",
			policy: core.DuplicatePolicyError,
		},
		{
			name:   "unknown-policy",
			policy: "Ignore",
			expected: field.ErrorList{
				field.NotSupported(
					field.NewPath("processing", "duplicatePolicy"),
					core.DuplicatePolicy("Ignore"),
					[]core.DuplicatePolicy{core.DuplicatePolicyMerge, core.DuplicatePolicyError},
				),
			},
		},
		{
			name:   "lowercase-policy",
			policy: "merge",
			expected: field.ErrorList{
				field.NotSupported(
					field.NewPath("processing", "duplicatePolicy"),
					core.DuplicatePolicy("merge"),
					[]core.DuplicatePolicy{core.DuplicatePolicyMerge, core.DuplicatePolicyError},
				),
			},
		},
		{
			name: "empty-policy",
			expected: field.ErrorList{
				field.NotSupported(
					field.NewPath("processing", "duplicatePolicy"),
					core.DuplicatePolicy(""),
					[]core.DuplicatePolicy{core.DuplicatePolicyMerge, core.DuplicatePolicyError},
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Processing.DuplicatePolicy = test.policy
			errs := validateConfigDuplicatePolicy(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
		})
	}
}

func TestValidateConfigShortEntryThreshold(t *testing.T) {
	for _, test := range []struct {
		name      string
		threshold *metav1.Duration
		expected  field.ErrorList
	}{
		{
			name: "unspecified",
		},
		{
			name:      "zero",
			threshold: &metav1.Duration{},
		},
		{
			name:      "positive",
			threshold: &metav1.Duration{Duration: time.Second},
		},
		{
			name:      "negative-nanosecond",
			threshold: &metav1.Duration{Duration: -time.Nanosecond},
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("processing", "shortEntryThreshold"),
					"-1ns",
					"must be greater than or equal to 0s",
				),
			},
		},
		{
			name:      "negative-minute",
			threshold: &metav1.Duration{Duration: -time.Minute},
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("processing", "shortEntryThreshold"),
					"-1m0s",
					"must be greater than or equal to 0s",
				),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := withValidConfig()
			cfg.Processing.ShortEntryThreshold = test.threshold
			errs := validateConfigShortEntryThreshold(cfg)
			errorCountMismatch := len(errs) != len(test.expected)
			hasErrorsToCompare := !errorCountMismatch && len(errs) != 0
			errorDetailsMismatch := hasErrorsToCompare && !reflect.DeepEqual(errs, test.expected)
			if errorCountMismatch || errorDetailsMismatch {
				t.Fatalf("want errors %#v, got %#v", test.expected, errs)
			}
		})
	}
}

func TestOnUpdateConfig(t *testing.T) {
	t.Run("ok", func(in *testing.T) {
		oldObj := withValidConfig()
		newObj := oldObj.DeepCopy()
		newObj.InputDirectories = []core.Directory{
			{Path: "./test"},
		}

		errs := OnUpdateConfig(oldObj, newObj)
		if len(errs) != 0 {
			t.Errorf("got error: %v", errs.ToAggregate())
		}
	})
}

func TestConfigDeprecations(t *testing.T) {
	t.Run("ok", func(in *testing.T) {
		ConfigDeprecations(withValidConfig())
	})
}
