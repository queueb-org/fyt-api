// Package validation checks FYT configuration and time entries.
package validation

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"fyt.queueb.org/api/core"
)

// maxMonetaryPrecision bounds the number of fractional digits in invoice totals.
const maxMonetaryPrecision int32 = 6

var allowedDuplicatePolicyMerge = []core.DuplicatePolicy{
	core.DuplicatePolicyMerge,
	core.DuplicatePolicyError,
}

var allowedTaxTypes = []core.TaxType{
	core.TaxTypeNone,
	core.TaxTypePercentage,
}

var nonNegativeDecimalPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// ConfigValidationOptions selects the document settings to validate.
type ConfigValidationOptions struct {
	Reports bool
	Invoice bool
}

// OnCreateConfig validates every section of a Config on create.
func OnCreateConfig(entry *core.Config) field.ErrorList {
	return ValidateConfig(entry, ConfigValidationOptions{Reports: true, Invoice: true})
}

// ValidateConfig validates shared settings and the selected document settings.
func ValidateConfig(entry *core.Config, options ConfigValidationOptions) (errorList field.ErrorList) {
	errorList = append(errorList, validateConfigProcessing(entry)...)
	errorList = append(errorList, validateConfigInputDirectories(entry)...)
	if options.Reports {
		errorList = append(errorList, validateConfigReports(entry)...)
	}
	if options.Invoice {
		errorList = append(errorList, ValidateInvoiceConfig(entry)...)
	}
	return append(errorList, validateConfigDocuments(entry)...)
}

// validateConfigReports centralizes validation of report settings,
// collecting all report configuration errors.
func validateConfigReports(entry *core.Config) (errorList field.ErrorList) {
	for _, verifier := range []func(entry *core.Config) (errorList field.ErrorList){
		validateConfigMemberName,
	} {
		errorList = append(errorList, verifier(entry)...)
	}
	return
}

// validateConfigProcessing centralizes duplicate policy and warning threshold
// validation, collecting all processing configuration errors.
func validateConfigProcessing(entry *core.Config) (errorList field.ErrorList) {
	for _, verifier := range []func(entry *core.Config) (errorList field.ErrorList){
		validateConfigDuplicatePolicy,
		validateConfigShortEntryThreshold,
	} {
		errorList = append(errorList, verifier(entry)...)
	}
	return
}

// validateConfigDocuments centralizes validation of output and template settings,
// collecting all document configuration errors.
func validateConfigDocuments(entry *core.Config) (errorList field.ErrorList) {
	for _, verifier := range []func(entry *core.Config) (errorList field.ErrorList){
		validateConfigOutputDirectory,
		validateConfigTemplatesDirectories,
	} {
		errorList = append(errorList, verifier(entry)...)
	}
	return
}

// validateConfigTemplatesDirectories requires a nonblank path for each custom
// template directory. An empty list selects the bundled template set.
func validateConfigTemplatesDirectories(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	directoriesPath := field.NewPath("documents", "templatesDirectories")
	for i, directory := range entry.Documents.TemplatesDirectories {
		if strings.TrimSpace(directory.Path) == "" {
			errorList = append(errorList, field.Required(
				directoriesPath.Index(i).Child("path"),
				"must not be empty or contain only whitespace",
			))
		}
	}
	return errorList
}

// validateConfigOutputDirectory requires a nonblank template for the PDF output
// path. The template is preserved as given for execution by the CLI.
func validateConfigOutputDirectory(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	if strings.TrimSpace(entry.Documents.OutputDirectory) == "" {
		errorList = append(errorList, field.Required(
			field.NewPath("documents", "outputDirectory"),
			"must not be empty or contain only whitespace",
		))
	}
	return errorList
}

// ValidateInvoiceConfig validates the invoice section of a Config.
func ValidateInvoiceConfig(entry *core.Config) (errorList field.ErrorList) {
	for _, verifier := range []func(entry *core.Config) (errorList field.ErrorList){
		validateConfigSupplierName,
		validateConfigCustomerName,
		validateConfigServiceDescription,
		validateConfigCurrency,
		validateConfigHourlyRate,
		validateConfigMonetaryPrecision,
		validateConfigTaxType,
		validateConfigTaxRatePercent,
		validateConfigAdditionalInvoiceLines,
	} {
		errorList = append(errorList, verifier(entry)...)
	}
	return
}

func validateConfigAdditionalInvoiceLines(entry *core.Config) (errorList field.ErrorList) {
	linesPath := field.NewPath("invoice", "additionalLines")
	for i, line := range entry.Invoice.AdditionalLines {
		path := linesPath.Index(i)
		if strings.TrimSpace(line.Description) == "" {
			errorList = append(errorList, field.Required(path.Child("description"), "must not be empty or contain only whitespace"))
		}
		if !nonNegativeDecimalPattern.MatchString(line.Quantity) || strings.Trim(line.Quantity, "0.") == "" {
			errorList = append(errorList, field.Invalid(path.Child("quantity"), line.Quantity, "must be a positive decimal number"))
		}
		if !nonNegativeDecimalPattern.MatchString(line.UnitPrice) {
			errorList = append(errorList, field.Invalid(path.Child("unitPrice"), line.UnitPrice, "must be a non-negative decimal number"))
		}
	}
	return
}

// validateConfigTaxRatePercent checks decimal syntax without floating-point
// conversion so invoice tax calculations retain the exact percentage.
// For None, the percentage is unused and is not validated.
func validateConfigTaxRatePercent(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	if entry.Invoice.Tax.Type == core.TaxTypeNone {
		return errorList
	}
	if !nonNegativeDecimalPattern.MatchString(entry.Invoice.Tax.RatePercent) {
		errorList = append(errorList, field.Invalid(
			field.NewPath("invoice", "tax", "ratePercent"), entry.Invoice.Tax.RatePercent,
			"must be a non-negative decimal number, such as 0, 5 or 7.50",
		))
	}
	return errorList
}

// validateConfigTaxType rejects unknown tax regimes so the invoice uses an
// explicitly supported method of calculating tax.
func validateConfigTaxType(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	if !slices.Contains(allowedTaxTypes, entry.Invoice.Tax.Type) {
		errorList = append(errorList, field.NotSupported(
			field.NewPath("invoice", "tax", "type"), entry.Invoice.Tax.Type,
			allowedTaxTypes,
		))
	}
	return errorList
}

// validateConfigMonetaryPrecision requires a bounded non-negative precision to
// keep monetary formatting practical while allowing explicit integer totals.
func validateConfigMonetaryPrecision(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	precisionPath := field.NewPath("invoice", "monetaryPrecision")
	precision := entry.Invoice.MonetaryPrecision
	if precision == nil {
		errorList = append(errorList, field.Required(precisionPath, "must be specified"))
	} else if *precision < 0 || *precision > maxMonetaryPrecision {
		errorList = append(errorList, field.Invalid(
			precisionPath, *precision,
			fmt.Sprintf("must be between 0 and %d, inclusive", maxMonetaryPrecision),
		))
	}
	return errorList
}

// validateConfigHourlyRate checks decimal syntax without floating-point conversion,
// preserving the exact rate and its precision for invoice calculations.
func validateConfigHourlyRate(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	if !nonNegativeDecimalPattern.MatchString(entry.Invoice.HourlyRate) {
		errorList = append(errorList, field.Invalid(
			field.NewPath("invoice", "hourlyRate"), entry.Invoice.HourlyRate,
			"must be a non-negative decimal number, such as 0, 25 or 25.50",
		))
	}
	return errorList
}

// validateConfigCurrency requires a nonblank designation so invoice monetary
// values have an explicit currency.
func validateConfigCurrency(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	currency := entry.Invoice.Currency
	if currency == nil || strings.TrimSpace(*currency) == "" {
		errorList = append(errorList, field.Required(
			field.NewPath("invoice", "currency"),
			"must not be empty or contain only whitespace",
		))
	}
	return errorList
}

// validateConfigServiceDescription requires nonblank text so the invoice
// describes the service being billed.
func validateConfigServiceDescription(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	if strings.TrimSpace(entry.Invoice.ServiceDescription) == "" {
		errorList = append(errorList, field.Required(
			field.NewPath("invoice", "serviceDescription"),
			"must not be empty or contain only whitespace",
		))
	}
	return errorList
}

// validateConfigSupplierName requires a nonblank name so the invoice identifies
// its supplier independently of the report member and customer.
func validateConfigSupplierName(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	if strings.TrimSpace(entry.Invoice.Supplier.Name) == "" {
		errorList = append(errorList, field.Required(
			field.NewPath("invoice", "supplier", "name"),
			"must not be empty or contain only whitespace",
		))
	}
	return errorList
}

// validateConfigCustomerName requires a nonblank name so the invoice identifies
// its customer independently of the supplier.
func validateConfigCustomerName(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	customer := entry.Invoice.Customer
	if customer == nil || strings.TrimSpace(customer.Name) == "" {
		errorList = append(errorList, field.Required(
			field.NewPath("invoice", "customer", "name"),
			"must not be empty or contain only whitespace",
		))
	}
	return errorList
}

// validateConfigMemberName requires a nonblank display name so both reports
// identify their member independently of the invoice parties.
func validateConfigMemberName(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	if strings.TrimSpace(entry.Reports.MemberName) == "" {
		errorList = append(errorList, field.Required(
			field.NewPath("reports", "memberName"),
			"must not be empty or contain only whitespace",
		))
	}
	return errorList
}

// validateConfigInputDirectories rejects blank paths so every configured input
// directory identifies a filesystem location. Nonblank paths are preserved as given.
func validateConfigInputDirectories(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	directoriesPath := field.NewPath("inputDirectories")
	for i, directory := range entry.InputDirectories {
		if strings.TrimSpace(directory.Path) == "" {
			errorList = append(errorList, field.Required(
				directoriesPath.Index(i).Child("path"),
				"must not be empty or contain only whitespace",
			))
		}
	}
	return errorList
}

// validateConfigDuplicatePolicy rejects unknown policies because they have no
// defined duplicate accounting semantics.
func validateConfigDuplicatePolicy(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	processingPath := field.NewPath("processing")
	if !slices.Contains(allowedDuplicatePolicyMerge, entry.Processing.DuplicatePolicy) {
		errorList = append(errorList, field.NotSupported(
			processingPath.Child("duplicatePolicy"), entry.Processing.DuplicatePolicy,
			allowedDuplicatePolicyMerge,
		))
	}
	return errorList
}

// validateConfigShortEntryThreshold rejects negative thresholds, which would
// unintentionally suppress warnings. Zero is allowed to disable warnings explicitly.
func validateConfigShortEntryThreshold(entry *core.Config) field.ErrorList {
	errorList := field.ErrorList{}
	processingPath := field.NewPath("processing")
	if threshold := entry.Processing.ShortEntryThreshold; threshold != nil && threshold.Duration < 0 {
		errorList = append(errorList, field.Invalid(
			processingPath.Child("shortEntryThreshold"), threshold.Duration.String(),
			"must be greater than or equal to 0s",
		))
	}
	return errorList
}

// OnUpdateConfig runs validation over [core.Config] object
// on update event.
func OnUpdateConfig(oldObj, newObj *core.Config) (errorList field.ErrorList) {
	// uncomment to update 'on update' validation.
	// if !reflect.DeepEqual(newObj.Spec.Options, oldObj.Spec.Options) {
	//      errorList = append(errorList,
	//              field.Invalid(field.NewPath(".spec.options"), newObj.Spec.Options, "the field is immutable"),
	//      )
	// }

	errorList = append(errorList, OnCreateConfig(newObj)...)
	return
}

// ConfigDeprecations is used to file warnings on create, update events.
func ConfigDeprecations(obj *core.Config) (out []string) {
	// uncomment to enable deprecations
	// if len(obj.Spec.Domain) != 0 {
	//      out = append(out,
	//              ".spec.domain "+
	//                      "field is deprecated and will be removed.",
	//      )
	// }

	return
}
