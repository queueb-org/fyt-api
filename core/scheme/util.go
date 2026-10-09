package scheme

import (
	internal "fyt.queueb.org/api/core"
	"fyt.queueb.org/api/core/v1alpha1"
)

// DefaultConfig returns default configuration entry.
func DefaultConfig() *internal.Config {
	versioned := &v1alpha1.Config{} // use default version
	Scheme.Default(versioned)
	cfg := &internal.Config{}
	_ = Scheme.Convert(versioned, cfg, nil)
	return cfg
}
