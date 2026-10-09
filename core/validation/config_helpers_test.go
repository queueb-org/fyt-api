package validation

import (
	"time"

	"fyt.queueb.org/api/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func withValidConfig() *core.Config {
	return &core.Config{
		Documents: core.DocumentsConfig{OutputDirectory: "./documents"},
		Reports:   core.ReportsConfig{MemberName: "John Doe"},
		Invoice: core.InvoiceSettings{
			Supplier:           core.InvoiceParty{Name: "John Doe"},
			Customer:           &core.InvoiceParty{Name: "ACME Inc"},
			ServiceDescription: "Software development",
			Currency:           new("USD"),
			HourlyRate:         "25",
			MonetaryPrecision:  new(int32(2)),
			Tax:                core.InvoiceTax{Type: core.TaxTypeNone, RatePercent: "0"},
		},
		Processing: core.ProcessingConfig{
			DuplicatePolicy:     core.DuplicatePolicyMerge,
			ShortEntryThreshold: &metav1.Duration{Duration: 5 * time.Minute},
		},
	}
}
