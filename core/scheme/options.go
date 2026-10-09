package scheme

import "k8s.io/apimachinery/pkg/runtime/serializer"

// Option configures decoding. The zero value enables strict decoding.
type Option struct {
	// UseStrict overrides strict decoding when set.
	UseStrict *bool
}

func mergeOptions(opts ...*Option) *Option {
	def := &Option{
		UseStrict: new(true),
	}

	for _, o := range opts {
		if o != nil && o.UseStrict != nil {
			def.UseStrict = o.UseStrict
		}
	}

	return def
}

// OptNoStrict enforces strict mode: off.
func OptNoStrict() *Option {
	return &Option{UseStrict: new(false)}
}

// Codecs returns the selected codec factory. A nil or zero Option uses Strict.
func (o *Option) Codecs() *serializer.CodecFactory {
	if o == nil || o.UseStrict == nil {
		return Strict
	}
	if *o.UseStrict {
		return Strict
	}
	return Unstrict
}
