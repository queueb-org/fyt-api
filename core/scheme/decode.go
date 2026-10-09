package scheme

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"

	yamlstream "go.yaml.in/yaml/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	internal "fyt.queueb.org/api/core"
	"fyt.queueb.org/api/core/v1alpha1"
)

var (
	// ErrLoad is returned if any error occurred during loading configuration file.
	ErrLoad = errors.New("could not load configuration file")
	// ErrDecode is returned if there was an error during object decoding operations.
	ErrDecode = errors.New("could not decode object")
)

// Decode decodes a JSON or YAML configuration, applies defaults for its
// declared API version, and converts it to the internal Config type.
//
// The document must contain a registered apiVersion and kind. Strict decoding
// rejects unknown and duplicate fields by default. Pass [OptNoStrict] to allow
// them. For YAML streams, Decode reads the first document.
func Decode(r io.Reader, opts ...*Option) (*internal.Config, error) {
	return decodeConfig[*internal.Config](r, nil, opts...)
}

// DecodeVersionedConfig decodes a registered configuration, applies defaults for
// its declared version, and returns it in the v1alpha1 API version.
func DecodeVersionedConfig(r io.Reader, opts ...*Option) (*v1alpha1.Config, error) {
	return decodeConfig[*v1alpha1.Config](r, []schema.GroupVersion{schema.GroupVersion(v1alpha1.GroupVersion)}, opts...)
}

func decodeConfig[T runtime.Object](r io.Reader, versions []schema.GroupVersion, opts ...*Option) (T, error) {
	opt := mergeOptions(opts...)
	var zero T

	data, err := io.ReadAll(r)
	if err != nil {
		return zero, fmt.Errorf("%w: could not read object: %w", ErrDecode, err)
	}

	return decodeObject[T](data, opt.Codecs().UniversalDecoder(versions...), "could not read configuration: ")
}

func decodeObject[T runtime.Object](data []byte, decoder runtime.Decoder, decodeErrorPrefix string) (T, error) {
	var zero T
	obj, gvk, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return zero, fmt.Errorf("%w: %s%w", ErrDecode, decodeErrorPrefix, err)
	}
	typed, ok := obj.(T)
	if !ok {
		return zero, fmt.Errorf("%w: unexpected object %s, expected %T", ErrDecode, gvk, zero)
	}
	return typed, nil
}

// Load reads a configuration file using [Decode].
func Load(path string, opts ...*Option) (*internal.Config, error) {
	return loadConfig(path, Decode, opts...)
}

// LoadVersionedConfig loads and decodes a configuration into the v1alpha1 API version.
func LoadVersionedConfig(path string, opts ...*Option) (*v1alpha1.Config, error) {
	return loadConfig(path, DecodeVersionedConfig, opts...)
}

func loadConfig[T runtime.Object](path string, decode func(io.Reader, ...*Option) (T, error), opts ...*Option) (config T, err error) {
	var fd *os.File
	if fd, err = os.Open(path); err != nil {
		return config, fmt.Errorf("%w: read configuration %w", ErrLoad, err)
	}
	defer fd.Close()

	if config, err = decode(fd, opts...); err != nil {
		err = fmt.Errorf("%w: decode configuration failed: %w", ErrLoad, err)
	}

	return
}

// DecodeVersionedInvoiceConfig reads one strict JSON or YAML InvoiceConfig.
// It preserves explicitly empty values and does not apply invoice defaults.
func DecodeVersionedInvoiceConfig(r io.Reader) (*v1alpha1.InvoiceConfig, error) {
	return decodeInvoiceDocument[*v1alpha1.InvoiceConfig](r)
}

// DecodeVersionedInvoiceConfigList reads one strict JSON or YAML InvoiceConfigList.
// It preserves item order and applies no invoice defaults to its items.
func DecodeVersionedInvoiceConfigList(r io.Reader) (*v1alpha1.InvoiceConfigList, error) {
	return decodeInvoiceDocument[*v1alpha1.InvoiceConfigList](r)
}

// DecodeInvoiceSettingsOverride reads a strict JSON or YAML document with only
// an invoice section, preserving explicitly empty values.
func DecodeInvoiceSettingsOverride(r io.Reader) (*v1alpha1.InvoiceSettingsOverride, error) {
	data, document, err := readInvoiceDocument(r)
	if err != nil {
		return nil, err
	}
	if _, ok := document["invoice"]; !ok {
		return nil, fmt.Errorf("%w: invoice: must be specified", ErrDecode)
	}
	for _, key := range slices.Sorted(maps.Keys(document)) {
		if key != "invoice" {
			return nil, fmt.Errorf("%w: unknown field %q", ErrDecode, key)
		}
	}
	if err := validateInvoiceValues(document, ""); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecode, err)
	}

	var config v1alpha1.InvoiceConfig
	if err := runtime.DecodeInto(Strict.UniversalDeserializer(), data, &config); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecode, err)
	}
	return &config.Invoice, nil
}

func decodeInvoiceDocument[T runtime.Object](r io.Reader) (T, error) {
	var zero T
	data, document, err := readInvoiceDocument(r)
	if err != nil {
		return zero, err
	}
	if err := validateInvoiceDocument(document); err != nil {
		return zero, fmt.Errorf("%w: %w", ErrDecode, err)
	}

	version := schema.GroupVersion(v1alpha1.GroupVersion)
	return decodeObject[T](data, Strict.UniversalDecoder(version), "")
}

func readInvoiceDocument(r io.Reader) ([]byte, map[string]any, error) {
	data, err := readInvoiceJSON(r)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrDecode, err)
	}
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrDecode, err)
	}
	return data, document, nil
}

func readInvoiceJSON(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if json.Valid(data) {
		return data, nil
	}

	// YAMLToJSONStrict checks keys but only reads the first YAML document.
	stream := yamlstream.NewDecoder(bytes.NewReader(data))
	var document any
	if err := stream.Decode(&document); err != nil {
		return nil, err
	}
	var extra any
	err = stream.Decode(&extra)
	if err == nil {
		return nil, errors.New("expected exactly one YAML document")
	}
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	if document != nil {
		if err := validateInvoiceValues(document, ""); err != nil {
			return nil, err
		}
		mapping, _ := document.(map[any]any)
		if err := validateYAMLInvoicePrecisions(mapping); err != nil {
			return nil, err
		}
	}
	return yaml.YAMLToJSONStrict(data)
}

func validateYAMLInvoicePrecisions(document map[any]any) error {
	if err := validateYAMLInvoicePrecision(document["invoice"], "invoice.monetaryPrecision"); err != nil {
		return err
	}
	items, _ := document["items"].([]any)
	for i, value := range items {
		item, _ := value.(map[any]any)
		path := fmt.Sprintf("items[%d].invoice.monetaryPrecision", i)
		if err := validateYAMLInvoicePrecision(item["invoice"], path); err != nil {
			return err
		}
	}
	return nil
}

func validateYAMLInvoicePrecision(value any, path string) error {
	settings, _ := value.(map[any]any)
	// YAML-to-JSON conversion can emit a rounded YAML float as a JSON integer.
	if _, isFloat := settings["monetaryPrecision"].(float64); isFloat {
		return fmt.Errorf("%s: must be an integer", path)
	}
	return nil
}

func validateInvoiceDocument(document map[string]any) error {
	if document == nil {
		return errors.New("invoice configuration document must be an object")
	}
	if err := validateInvoiceValues(document, ""); err != nil {
		return err
	}
	switch document["kind"] {
	case "InvoiceConfig":
		if _, ok := document["invoice"]; !ok {
			return errors.New("invoice: must be specified")
		}
	case "InvoiceConfigList":
		items, _ := document["items"].([]any)
		version := v1alpha1.GroupVersion.String()
		for i, value := range items {
			item, ok := value.(map[string]any)
			if !ok {
				continue // The typed decoder rejects invalid item types.
			}
			if _, ok := item["invoice"]; !ok {
				return fmt.Errorf("items[%d].invoice: must be specified", i)
			}
			if kind, specified := item["kind"]; specified && kind != "InvoiceConfig" {
				return fmt.Errorf("items[%d].kind: must be InvoiceConfig", i)
			}
			if apiVersion, specified := item["apiVersion"]; specified && apiVersion != version {
				return fmt.Errorf("items[%d].apiVersion: must be %s", i, version)
			}
		}
	}
	return nil
}

func validateInvoiceValues(value any, path string) error {
	switch value := value.(type) {
	case nil:
		return fmt.Errorf("%s: null is not allowed", path)
	case map[any]any:
		// YAML mappings can have keys that would collide after string conversion.
		fields := make(map[string]any, len(value))
		for key, field := range value {
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("%s: mapping key %v must be a string", path, key)
			}
			fields[name] = field
		}
		return validateInvoiceValues(fields, path)
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(value)) {
			child := key
			if path != "" {
				child = path + "." + key
			}
			if err := validateInvoiceValues(value[key], child); err != nil {
				return err
			}
		}
	case []any:
		for i, item := range value {
			if err := validateInvoiceValues(item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}
