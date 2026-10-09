// Package scheme provides the runtime scheme and codecs for all supported
// versions of the FYT API.
package scheme

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	internal "fyt.queueb.org/api/core"
	"fyt.queueb.org/api/core/v1alpha1"
)

var (
	// Scheme contains the internal configuration API and every supported wire
	// version.
	Scheme = runtime.NewScheme()

	// Strict performs strict decoding, defaulting, and conversion between the
	// registered configuration API versions.
	//
	// Use [DecodeVersionedInvoiceConfig] or [DecodeVersionedInvoiceConfigList]
	// for invoice documents. Those functions require one document with the
	// expected invoice structure and reject nulls and mismatched list item
	// type metadata. They also check YAML monetary precision before converting
	// to JSON.
	Strict = new(serializer.NewCodecFactory(Scheme, serializer.EnableStrict))
	// Unstrict performs decoding with strict disabled.
	Unstrict = new(serializer.NewCodecFactory(Scheme))
)

func init() {
	utilruntime.Must(internal.Install(Scheme))
	utilruntime.Must(v1alpha1.Install(Scheme))

	utilruntime.Must(
		Scheme.SetVersionPriority(schema.GroupVersion(v1alpha1.GroupVersion)),
	)
}
