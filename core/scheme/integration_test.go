package scheme

import (
	"bytes"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"fyt.queueb.org/api/core"
	"fyt.queueb.org/api/core/v1alpha1"
)

// TestIntegration_InvoiceConfig_RoundTrip checks that the API codecs and scheme
// preserve field presence through internal conversion, copying, and JSON/YAML
// serialization for a single invoice configuration and for an ordered list.
func TestIntegration_InvoiceConfig_RoundTrip(t *testing.T) {
	first := core.InvoiceConfig{
		Invoice: core.InvoiceSettingsOverride{
			Customer: &core.InvoicePartyOverride{
				Email:   new("billing@example.org"),
				Options: map[string]string{},
			},
			HourlyRate:          new("0"),
			MonetaryPrecision:   new(int32(0)),
			PaymentInstructions: new(""),
			Tax:                 &core.InvoiceTaxOverride{},
			AdditionalLines:     []core.InvoiceAdditionalLine{},
		},
	}
	second := core.InvoiceConfig{
		Invoice: core.InvoiceSettingsOverride{
			Supplier: &core.InvoicePartyOverride{
				Options: map[string]string{"customer_id": "42"},
			},
			AdditionalLines: []core.InvoiceAdditionalLine{
				{Description: "License", Quantity: "1", UnitPrice: "9.95"},
			},
		},
	}
	configList := &core.InvoiceConfigList{Items: []core.InvoiceConfig{first, second, {}}}

	for _, mediaType := range []string{runtime.ContentTypeJSON, runtime.ContentTypeYAML} {
		t.Run(mediaType, func(t *testing.T) {
			info, ok := runtime.SerializerInfoForMediaType(Strict.SupportedMediaTypes(), mediaType)
			if !ok {
				t.Fatalf("no serializer for %s", mediaType)
			}
			version := schema.GroupVersion(v1alpha1.GroupVersion)
			encoder := Strict.EncoderForVersion(info.Serializer, version)
			for _, tc := range []struct {
				kind   string
				object runtime.Object
			}{
				{kind: "InvoiceConfig", object: &first},
				{kind: "InvoiceConfigList", object: configList},
			} {
				t.Run(tc.kind, func(t *testing.T) {
					data, err := runtime.Encode(encoder, tc.object.DeepCopyObject())
					if err != nil {
						t.Fatal(err)
					}
					var decoded runtime.Object
					if tc.kind == "InvoiceConfig" {
						decoded, err = DecodeVersionedInvoiceConfig(bytes.NewReader(data))
					} else {
						decoded, err = DecodeVersionedInvoiceConfigList(bytes.NewReader(data))
					}
					if err != nil {
						t.Fatalf("could not read %s: %v\n%s", tc.kind, err, data)
					}
					if got := decoded.GetObjectKind().GroupVersionKind(); got != version.WithKind(tc.kind) {
						t.Fatalf("unexpected kind: %v", got)
					}
					actual := tc.object.DeepCopyObject()
					if err := Scheme.Convert(decoded, actual, nil); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(actual, tc.object) {
						t.Fatalf("field presence or values changed after round trip\nencoded: %s\n got %+v\nwant %+v", data, actual, tc.object)
					}
				})
			}
		})
	}

	copyList := configList.DeepCopy()
	*copyList.Items[0].Invoice.Customer.Email = "other@example.org"
	copyList.Items[1].Invoice.Supplier.Options["customer_id"] = "23"
	copyList.Items[1].Invoice.AdditionalLines[0].Quantity = "2"
	if got := *configList.Items[0].Invoice.Customer.Email; got != "billing@example.org" {
		t.Errorf("copy mutation changed the original customer email to %q", got)
	}
	if got := configList.Items[1].Invoice.Supplier.Options["customer_id"]; got != "42" {
		t.Errorf("copy mutation changed the original supplier option to %q", got)
	}
	if got := configList.Items[1].Invoice.AdditionalLines[0].Quantity; got != "1" {
		t.Errorf("copy mutation changed the original additional line quantity to %q", got)
	}
}
