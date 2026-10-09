package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// DefaultMemberName is the report member name used when no name is specified.
	DefaultMemberName = "John Doe"
	// DefaultSupplierName is the invoice supplier name used when no name is specified.
	DefaultSupplierName = "John Doe"
	// DefaultCustomerName is the invoice customer name used when the customer is unspecified.
	DefaultCustomerName = "ACME Inc"
	// DefaultServiceDescription is the invoice service description used when unspecified.
	DefaultServiceDescription = "Software development"
	// DefaultCurrency is the invoice currency used when unspecified.
	DefaultCurrency = "USD"
	// DefaultHourlyRate is the invoice hourly rate used when unspecified.
	DefaultHourlyRate = "25"
	// DefaultMonetaryPrecision is the number of fractional digits in invoice totals.
	DefaultMonetaryPrecision int32 = 2
	// DefaultTaxRatePercent is the former default invoice tax percentage.
	//
	// Deprecated: Set RatePercent explicitly when selecting Percentage.
	DefaultTaxRatePercent = "0"
	// DefaultOutputDirectory is the PDF output path template used when unspecified.
	DefaultOutputDirectory = "./documents/{{ .YearMonth }}"
	// DefaultShortEntryThreshold keeps threshold metric (5 minutes by default).
	DefaultShortEntryThreshold = time.Minute * 5
)

func init() {
	localSchemeBuilder.Register(addDefaultingFuncs)
}

// addDefaultingFuncs registers generated defaulting functions with the scheme.
func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_TimeEntry sets defaults for [TimeEntry] object.
func SetDefaults_TimeEntry(in *TimeEntry) {
	//lint:ignore S1023, intended.
	return
}

// SetDefaults_Config sets defaults for [Config] object.
func SetDefaults_Config(in *Config) {
	setConfigReports(in)
	setConfigInvoice(in)
	setConfigProcessing(in)
	setConfigDocuments(in)
}

// setConfigDocuments supplies defaults for unspecified document settings.
func setConfigDocuments(in *Config) {
	if in.Documents.OutputDirectory == "" {
		in.Documents.OutputDirectory = DefaultOutputDirectory
	}
}

// setConfigReports supplies defaults for unspecified report settings.
func setConfigReports(in *Config) {
	if in.Reports.MemberName == nil {
		in.Reports.MemberName = new(DefaultMemberName)
	}
}

// setConfigInvoice supplies defaults for unspecified invoice settings.
func setConfigInvoice(in *Config) {
	if in.Invoice.Supplier.Name == "" {
		in.Invoice.Supplier.Name = DefaultSupplierName
	}
	if in.Invoice.Customer == nil {
		in.Invoice.Customer = &InvoiceParty{Name: DefaultCustomerName}
	}
	if in.Invoice.ServiceDescription == "" {
		in.Invoice.ServiceDescription = DefaultServiceDescription
	}
	if in.Invoice.Currency == nil {
		in.Invoice.Currency = new(DefaultCurrency)
	}
	if in.Invoice.HourlyRate == "" {
		in.Invoice.HourlyRate = DefaultHourlyRate
	}
	if in.Invoice.MonetaryPrecision == nil {
		in.Invoice.MonetaryPrecision = new(DefaultMonetaryPrecision)
	}
	if in.Invoice.Tax.Type == "" {
		in.Invoice.Tax.Type = TaxTypeNone
	}
}

// setConfigProcessing supplies defaults while preserving explicit processing settings.
func setConfigProcessing(in *Config) {
	if in.Processing.DuplicatePolicy == "" {
		in.Processing.DuplicatePolicy = DuplicatePolicyMerge
	}
	if in.Processing.ShortEntryThreshold == nil {
		in.Processing.ShortEntryThreshold = &metav1.Duration{Duration: DefaultShortEntryThreshold}
	}
}
