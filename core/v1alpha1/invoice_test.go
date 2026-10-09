package v1alpha1

import (
	"reflect"
	"testing"

	"fyt.queueb.org/api/core"
)

func TestInvoiceConfigRoundTrip(t *testing.T) {
	scheme := WithNewLocalScheme(t)
	cfg := &Config{Invoice: InvoiceSettings{
		Supplier: InvoiceParty{
			Name: "Supplier", Address: "1 Main Street", City: "Example City",
			Phone: "0123", Email: "supplier@example.com", Options: map[string]string{"customer_id": "013"},
		},
		Customer: &InvoiceParty{
			Name: "Customer", Address: "2 Main Street", City: "Sample City",
			Phone: "0456", Email: "customer@example.com",
		},
		AdditionalLines: []InvoiceAdditionalLine{{Description: "License", Quantity: "2", UnitPrice: "10.000"}},
	}}

	versionedCopy := cfg.DeepCopy()
	versionedCopy.Invoice.Customer.Name = "Other customer"
	versionedCopy.Invoice.Supplier.Options["customer_id"] = "0"
	versionedCopy.Invoice.AdditionalLines[0].Description = "Other service"
	if cfg.Invoice.Supplier.Options["customer_id"] != "013" || cfg.Invoice.AdditionalLines[0].Description != "License" {
		t.Fatal("mutating the versioned copy changed the original invoice")
	}
	if cfg.Invoice.Customer.Name != "Customer" {
		t.Fatal("mutating the versioned copy changed the original customer")
	}

	var internal core.Config
	if err := scheme.Convert(cfg, &internal, nil); err != nil {
		t.Fatal(err)
	}
	internalCopy := internal.DeepCopy()
	internalCopy.Invoice.Customer.Name = "Other customer"
	internalCopy.Invoice.Supplier.Options["customer_id"] = "0"
	internalCopy.Invoice.AdditionalLines[0].Description = "Other service"
	if internal.Invoice.Supplier.Options["customer_id"] != "013" || internal.Invoice.AdditionalLines[0].Description != "License" {
		t.Fatal("mutating the internal copy changed the original invoice")
	}
	if internal.Invoice.Customer.Name != "Customer" {
		t.Fatal("mutating the internal copy changed the original customer")
	}

	var roundTrip Config
	if err := scheme.Convert(&internal, &roundTrip, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundTrip.Invoice, cfg.Invoice) {
		t.Errorf("invoice data changed during conversion: got %+v, want %+v", roundTrip.Invoice, cfg.Invoice)
	}
}
