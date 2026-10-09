package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TimeEntry keeps a time entry for evaluating reports.
type TimeEntry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Start period keeps date start period of entry, e.g. 2026-09-01 10:00
	Start metav1.Time `json:"start,omitzero"`
	// End keeps date end period, e.g. 2026-09-01 12:00
	End metav1.Time `json:"end,omitzero"`
	// Description keeps human readable description.
	Description string `json:"description,omitzero"`
	// Ticket is internal system ticket in a free form.
	Ticket string `json:"ticket,omitzero"`
	// Status keeps internal status for [TimeEntry] object.
	Status TimeEntryStatus `json:"status"`
}

// TimeEntryStatus keeps internal status for [TimeEntry] object.
// Conditions represented over `Ready` and `Failed` states.
// Ready and Failed should not have True status sim.
type TimeEntryStatus struct {
	// Conditions keep internal conditions for resource.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions" patchStrategy:"merge" patchMergeKey:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TimeEntryList represents [TimeEntry] objects over API.
type TimeEntryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Meta keeps different meta information in string format by keys.
	Meta  map[string]string `json:"meta,omitempty"`
	Items []TimeEntry       `json:"items,omitempty"`
}

// Directory identifies a filesystem directory with optional descriptive metadata.
type Directory struct {
	// Path is the filesystem path of the directory.
	Path string `json:"path,omitzero"`
	// Description keeps user description if it's required.
	Description string `json:"description,omitzero"`
}

// ReportsConfig contains settings shared by detailed and summary reports.
type ReportsConfig struct {
	// MemberName is the display name used in both reports, independent of Invoice.Supplier.
	// Defaults to John Doe when omitted or null. An explicit name must be nonblank.
	MemberName *string `json:"memberName,omitempty"`
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
	// Defaults to Merge when unspecified.
	DuplicatePolicy DuplicatePolicy `json:"duplicatePolicy,omitzero"`
	// ShortEntryThreshold is the warning threshold for a complete source entry.
	// Defaults to 5m when unspecified; an explicit zero disables warnings.
	ShortEntryThreshold *metav1.Duration `json:"shortEntryThreshold,omitempty"`
}

// InvoiceParty identifies an invoice party and carries optional contact and template data.
type InvoiceParty struct {
	Name    string            `json:"name,omitzero"`
	Details string            `json:"details,omitzero"`
	Address string            `json:"address,omitzero"`
	City    string            `json:"city,omitzero"`
	Phone   string            `json:"phone,omitzero"`
	Email   string            `json:"email,omitzero"`
	Options map[string]string `json:"options,omitempty"`
}

// InvoiceAdditionalLine describes an extra charge in the invoice currency.
type InvoiceAdditionalLine struct {
	Description string `json:"description"`
	// Quantity is an exact positive decimal, expressed as text.
	Quantity string `json:"quantity"`
	// UnitPrice is an exact non-negative decimal, expressed as text.
	UnitPrice string `json:"unitPrice"`
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
	// Defaults to None when unspecified.
	Type TaxType `json:"type,omitzero"`
	// RatePercent is an exact non-negative decimal percentage.
	// Required for Percentage. Uses digits with an optional fractional part, e.g. "7.50".
	// Unused and not validated when Type is None.
	RatePercent string `json:"ratePercent,omitzero"`
}

// InvoiceSettings contains reusable invoice settings.
type InvoiceSettings struct {
	// Supplier identifies the invoice supplier; its name defaults to John Doe.
	Supplier InvoiceParty `json:"supplier,omitzero"`
	// Customer identifies the invoice customer. An omitted or null customer defaults to ACME Inc.
	Customer *InvoiceParty `json:"customer,omitzero"`
	// ServiceDescription describes the billed service; defaults to Software development.
	ServiceDescription string `json:"serviceDescription,omitzero"`
	// Currency is the denomination of invoice monetary values.
	// Defaults to USD when omitted or null. A supplied value must be nonblank.
	Currency *string `json:"currency,omitempty"`
	// HourlyRate is an exact non-negative decimal amount expressed as text.
	// Uses digits with an optional fractional part, e.g. "25.50"; defaults to "25".
	HourlyRate string `json:"hourlyRate,omitzero"`
	// MonetaryPrecision is the number of fractional digits in totals, from 0 to 6 inclusive.
	// Defaults to 2 when omitted or null; an explicit zero is valid.
	MonetaryPrecision *int32     `json:"monetaryPrecision,omitempty"`
	Tax               InvoiceTax `json:"tax,omitzero"`
	// PaymentInstructions contains optional multiline payment details.
	PaymentInstructions string `json:"paymentInstructions,omitzero"`
	// AdditionalLines are billed in the order supplied, after any hourly service.
	AdditionalLines []InvoiceAdditionalLine `json:"additionalLines,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InvoiceConfig carries partial invoice settings for one document.
type InvoiceConfig struct {
	metav1.TypeMeta `json:",inline"`

	Invoice InvoiceSettingsOverride `json:"invoice"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InvoiceConfigList carries invoice configurations in order.
type InvoiceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []InvoiceConfig `json:"items,omitzero"`
}

// InvoiceSettingsOverride distinguishes absent fields from explicit values.
type InvoiceSettingsOverride struct {
	Supplier            *InvoicePartyOverride   `json:"supplier,omitempty"`
	Customer            *InvoicePartyOverride   `json:"customer,omitempty"`
	ServiceDescription  *string                 `json:"serviceDescription,omitempty"`
	Currency            *string                 `json:"currency,omitempty"`
	HourlyRate          *string                 `json:"hourlyRate,omitempty"`
	MonetaryPrecision   *int32                  `json:"monetaryPrecision,omitempty"`
	Tax                 *InvoiceTaxOverride     `json:"tax,omitempty"`
	PaymentInstructions *string                 `json:"paymentInstructions,omitempty"`
	AdditionalLines     []InvoiceAdditionalLine `json:"additionalLines,omitzero"`
}

// InvoicePartyOverride contains individual party fields. Options replaces the whole map.
type InvoicePartyOverride struct {
	Name    *string           `json:"name,omitempty"`
	Details *string           `json:"details,omitempty"`
	Address *string           `json:"address,omitempty"`
	City    *string           `json:"city,omitempty"`
	Phone   *string           `json:"phone,omitempty"`
	Email   *string           `json:"email,omitempty"`
	Options map[string]string `json:"options,omitzero"`
}

// InvoiceTaxOverride contains individual tax fields.
type InvoiceTaxOverride struct {
	Type        *TaxType `json:"type,omitempty"`
	RatePercent *string  `json:"ratePercent,omitempty"`
}

// DocumentsConfig selects output and template directories for PDF generation.
type DocumentsConfig struct {
	// OutputDirectory is a Go template for the final PDF output path; defaults to ./documents/{{ .YearMonth }}.
	OutputDirectory string `json:"outputDirectory,omitzero"`
	// TemplatesDirectories identifies custom template directories; empty selects the bundled set.
	TemplatesDirectories []Directory `json:"templatesDirectories,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Config keeps default configuration.
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	InputDirectories []Directory      `json:"inputDirectories,omitempty"`
	Reports          ReportsConfig    `json:"reports,omitzero"`
	Processing       ProcessingConfig `json:"processing,omitzero"`
	Invoice          InvoiceSettings  `json:"invoice,omitzero"`
	Documents        DocumentsConfig  `json:"documents,omitzero"`
}
