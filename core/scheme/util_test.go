package scheme

import "testing"

func TestDefaultConfig(t *testing.T) {
	t.Run("ok", func(in *testing.T) {
		cfg := DefaultConfig()
		if cfg == nil {
			in.Errorf("default config not nil expected.")
		}
	})
}
