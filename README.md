# FYT API

`fyt.queueb.org/api` is the Go module that defines FYT configuration and
timesheet exchange objects. It provides data types, codecs, conversion,
defaulting, and validation for use in Go programs.

## Install

Use Go 1.27.0 or later. Run this command from your Go module.

```sh
go get fyt.queueb.org/api@v0.1.0
```

## Packages

| Package | Purpose |
| --- | --- |
| `fyt.queueb.org/api/core/v1alpha1` | Types for the `fyt.queueb.org/v1alpha1` document format. |
| `fyt.queueb.org/api/core` | Common Go model used by conversion and validation. |
| `fyt.queueb.org/api/core/scheme` | Registered types, codecs, and helpers for reading configuration. |
| `fyt.queueb.org/api/core/validation` | Config and time entry validation, including helpers for versioned objects. |

Use `v1alpha1` types for serialized documents. The `apiVersion` and `kind`
fields identify the schema. The `fyt.queueb.org/v1alpha1` API defines
`Config`, `TimeEntry`, `TimeEntryList`, `InvoiceConfig`, and
`InvoiceConfigList`. The Go module version and the document schema version are
separate. The `v1alpha1` schema may change in later releases.

## Load and validate configuration

`scheme.Load` reads JSON or YAML, applies defaults for the declared version,
and converts the result to the common Go model. It rejects unknown and
duplicate fields by default. For a YAML stream, it reads the first document.
Validate the result before using it.
See the [example configuration](resources/config.example.yaml).

```go
package main

import (
	"log"

	"fyt.queueb.org/api/core/scheme"
	"fyt.queueb.org/api/core/validation"
)

func main() {
	config, err := scheme.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := validation.OnCreateConfig(config).ToAggregate(); err != nil {
		log.Fatal(err)
	}
	log.Print(config.Reports.MemberName)
}
```

Use `scheme.DecodeVersionedInvoiceConfig` and
`scheme.DecodeVersionedInvoiceConfigList` to read one invoice configuration
document. These helpers preserve explicitly supplied values and leave invoice
defaults to the caller.

## Development

Run `go test ./...` and `go vet ./...` from the module directory.

To regenerate registration, conversion, defaulting, and deepcopy code after
editing API types, install the pinned generator versions and run the helper
with those binaries on your `PATH`.

```sh
go install queueb.org/gena@v0.0.1
go install \
  k8s.io/code-generator/cmd/register-gen@v0.36.5 \
  k8s.io/code-generator/cmd/deepcopy-gen@v0.36.5 \
  k8s.io/code-generator/cmd/conversion-gen@v0.36.5 \
  k8s.io/code-generator/cmd/defaulter-gen@v0.36.5
./hack/run-gens.sh
```

Commit the generated files with the type changes.

Distributed under the [MIT License](LICENSE).
