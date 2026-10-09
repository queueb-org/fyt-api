package scheme

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func TestMergeOptions(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		if got := mergeOptions(nil, &Option{}); !*got.UseStrict {
			t.Error("expected strict decoding by default")
		}
	})
	t.Run("last explicit option wins", func(t *testing.T) {
		got := mergeOptions(&Option{UseStrict: new(true)}, OptNoStrict(), nil, &Option{})
		if *got.UseStrict {
			t.Error("expected explicit non-strict decoding to be preserved")
		}
	})
}

func TestOption_Codecs(t *testing.T) {
	testCases := []struct {
		name   string
		option *Option
		want   *serializer.CodecFactory
	}{
		{name: "nil option", want: Strict},
		{name: "zero option", option: &Option{}, want: Strict},
		{name: "strict", option: &Option{UseStrict: new(true)}, want: Strict},
		{name: "non-strict", option: OptNoStrict(), want: Unstrict},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.option.Codecs(); got != tc.want {
				t.Errorf("got codec factory %p, want %p", got, tc.want)
			}
		})
	}
}
