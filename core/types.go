package core

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TimeEntry keeps a time entry for evaluating reports.
type TimeEntry struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// Start period keeps date start period of entry, e.g. 2026-09-01 10:00
	Start metav1.Time
	// End keeps date end period, e.g. 2026-09-01 12:00
	End metav1.Time
	// Description keeps human readable description.
	Description string
	// Ticket is internal system ticket in a free form.
	Ticket string
	// Status keeps internal status for [TimeEntry] object.
	Status TimeEntryStatus
}

// TimeEntryStatus keeps internal status for [TimeEntry] object.
// Conditions represented over `Ready` and `Failed` states.
// Ready and Failed should not have True status sim.
type TimeEntryStatus struct {
	Conditions []metav1.Condition
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TimeEntryList represents [TimeEntry] objects over API.
type TimeEntryList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Meta  map[string]string
	Items []TimeEntry
}

// Directory identifies a filesystem directory with optional descriptive metadata.
type Directory struct {
	// Path is the filesystem path of the directory.
	Path string
	// Description keeps user description if it's required.
	Description string
}

// ReportsConfig contains settings shared by detailed and summary reports.
type ReportsConfig struct {
	// MemberName is the display name used in both reports, independent of Invoice.Supplier.
	// Defaults to John Doe when unspecified in the versioned configuration.
	MemberName string
}

// DuplicatePolicy selects how proven duplicate entries are handled.
type DuplicatePolicy string

const (
	// DuplicatePolicyMerge combines proven duplicates while retaining their sources.
	DuplicatePolicyMerge DuplicatePolicy = "Merge"
	// DuplicatePolicyError rejects proven duplicates.
	DuplicatePolicyError DuplicatePolicy = "Error"
)

// ProcessingConfig contains settings shared by validation, export and generation.
type ProcessingConfig struct {
	// DuplicatePolicy selects how proven duplicates are handled: Merge or Error.
	// Defaults to Merge when unspecified in the versioned configuration.
	DuplicatePolicy DuplicatePolicy
	// ShortEntryThreshold is the warning threshold for a complete source entry.
	// Defaults to 5m in the versioned configuration; an explicit zero disables warnings.
	ShortEntryThreshold *metav1.Duration
}

// InvoiceParty identifies an invoice party and carries optional contact and template data.
type InvoiceParty struct {
	Name    string
	Details string
	Address string
	City    string
	Phone   string
	Email   string
	Options map[string]string
}

// InvoiceAdditionalLine describes an extra charge in the invoice currency.
type InvoiceAdditionalLine struct {
	Description string
	Quantity    string
	UnitPrice   string
}

// TaxType identifies an invoice tax regime.
type TaxType string

const (
	// TaxTypeNone selects an invoice without tax.
	TaxTypeNone TaxType = "None"
	// TaxTypePercentage selects a percentage tax on the subtotal.
	TaxTypePercentage TaxType = "Percentage"
)

// InvoiceTax describes the invoice tax regime.
type InvoiceTax struct {
	// Type is None or Percentage.
	// Defaults to None when unspecified in the versioned configuration.
	Type TaxType
	// RatePercent is an exact non-negative decimal percentage.
	// Required for Percentage. Uses digits with an optional fractional part, e.g. "7.50".
	// Unused and not validated when Type is None.
	RatePercent string
}

// InvoiceSettings contains reusable invoice settings.
type InvoiceSettings struct {
	// Supplier identifies the invoice supplier; its name defaults to John Doe.
	Supplier InvoiceParty
	// Customer identifies the invoice customer after configuration defaulting.
	Customer *InvoiceParty
	// ServiceDescription describes the billed service; defaults to Software development.
	ServiceDescription string
	// Currency is the denomination of invoice monetary values.
	// Defaults to USD when omitted or null in the versioned base configuration.
	Currency *string
	// HourlyRate is an exact non-negative decimal amount expressed as text.
	// Uses digits with an optional fractional part, e.g. "25.50"; defaults to "25".
	HourlyRate string
	// MonetaryPrecision is the number of fractional digits in totals, from 0 to 6 inclusive.
	// Defaults to 2 in the versioned configuration; an explicit zero is valid.
	MonetaryPrecision *int32
	Tax               InvoiceTax
	// PaymentInstructions contains optional multiline payment details.
	PaymentInstructions string
	// AdditionalLines are billed in the order supplied, after any hourly service.
	AdditionalLines []InvoiceAdditionalLine
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InvoiceConfig carries partial invoice settings for one document.
type InvoiceConfig struct {
	metav1.TypeMeta

	Invoice InvoiceSettingsOverride
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InvoiceConfigList carries invoice configurations in order.
type InvoiceConfigList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []InvoiceConfig
}

// InvoiceSettingsOverride distinguishes absent fields from explicit values.
type InvoiceSettingsOverride struct {
	Supplier            *InvoicePartyOverride
	Customer            *InvoicePartyOverride
	ServiceDescription  *string
	Currency            *string
	HourlyRate          *string
	MonetaryPrecision   *int32
	Tax                 *InvoiceTaxOverride
	PaymentInstructions *string
	AdditionalLines     []InvoiceAdditionalLine
}

// InvoicePartyOverride contains individual party fields. Options replaces the whole map.
type InvoicePartyOverride struct {
	Name    *string
	Details *string
	Address *string
	City    *string
	Phone   *string
	Email   *string
	Options map[string]string
}

// InvoiceTaxOverride contains individual tax fields.
type InvoiceTaxOverride struct {
	Type        *TaxType
	RatePercent *string
}

// DocumentsConfig selects output and template directories for PDF generation.
type DocumentsConfig struct {
	// OutputDirectory is a Go template for the final PDF output path.
	OutputDirectory string
	// TemplatesDirectories identifies custom template directories; empty selects the bundled set.
	TemplatesDirectories []Directory
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Config keeps default configuration.
type Config struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	InputDirectories []Directory
	Reports          ReportsConfig
	Processing       ProcessingConfig
	Invoice          InvoiceSettings
	Documents        DocumentsConfig
}
