package validation

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"fyt.queueb.org/api/core"
	"fyt.queueb.org/api/core/v1alpha1"
)

func TestValidateVersionedCreate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		config := &v1alpha1.Config{}
		v1alpha1.SetDefaults_Config(config)
		errs, err := ValidateVersionedCreate(config, OnCreateConfig)
		if err != nil {
			t.Fatal(err)
		}
		if len(errs) != 0 {
			t.Errorf("expected a valid config, got %v", errs)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		config := &v1alpha1.Config{}
		v1alpha1.SetDefaults_Config(config)
		config.Reports.MemberName = new(" ")
		before := config.DeepCopy()

		errs, err := ValidateVersionedCreate(config, OnCreateConfig)
		if err != nil {
			t.Fatal(err)
		}
		if len(errs) != 1 {
			t.Fatalf("expected one validation error, got %v", errs)
		}
		if errs[0].Field != "reports.memberName" || errs[0].Type != field.ErrorTypeRequired {
			t.Errorf("expected reports.memberName to be required, got %v", errs[0])
		}
		if !reflect.DeepEqual(config, before) {
			t.Errorf("validation changed the config: %+v", config)
		}
	})

	t.Run("conversion-error", func(t *testing.T) {
		errs, err := ValidateVersionedCreate(&metav1.Status{}, func(*core.Config) field.ErrorList {
			t.Fatal("validator called after conversion failed")
			return nil
		})
		if err == nil || len(errs) != 0 {
			t.Errorf("expected a conversion error and no field violations, got %v, %v", err, errs)
		}
	})
}

func TestValidateVersionedUpdate(t *testing.T) {
	t.Run("passes-old-config-before-new-config", func(t *testing.T) {
		oldConfig := &v1alpha1.Config{Reports: v1alpha1.ReportsConfig{MemberName: new("Old name")}}
		newConfig := &v1alpha1.Config{Reports: v1alpha1.ReportsConfig{MemberName: new("New name")}}
		called := false
		errs, err := ValidateVersionedUpdate(oldConfig, newConfig, func(oldInternal, newInternal *core.Config) field.ErrorList {
			called = true
			if oldInternal.Reports.MemberName != "Old name" || newInternal.Reports.MemberName != "New name" {
				t.Errorf("expected old config first and new config second, got %q and %q",
					oldInternal.Reports.MemberName, newInternal.Reports.MemberName)
			}
			return nil
		})
		if err != nil || len(errs) != 0 {
			t.Fatalf("expected a successful update, got %v, %v", errs, err)
		}
		if !called {
			t.Error("expected the update validator to run")
		}
	})

	t.Run("returns-update-violations", func(t *testing.T) {
		oldConfig := &v1alpha1.Config{}
		v1alpha1.SetDefaults_Config(oldConfig)
		newConfig := oldConfig.DeepCopy()
		newConfig.Reports.MemberName = new(" ")

		errs, err := ValidateVersionedUpdate(oldConfig, newConfig, OnUpdateConfig)
		if err != nil {
			t.Fatal(err)
		}
		if len(errs) != 1 {
			t.Fatalf("expected one validation error, got %v", errs)
		}
		if errs[0].Field != "reports.memberName" || errs[0].Type != field.ErrorTypeRequired {
			t.Errorf("expected reports.memberName to be required, got %v", errs[0])
		}
	})

	for _, test := range []struct {
		name                 string
		oldObject, newObject runtime.Object
		errorPrefix          string
	}{
		{"old-conversion-fails", &metav1.Status{}, &v1alpha1.Config{}, "convert old object:"},
		{"new-conversion-fails", &v1alpha1.Config{}, &metav1.Status{}, "convert new object:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			errs, err := ValidateVersionedUpdate(test.oldObject, test.newObject, func(*core.Config, *core.Config) field.ErrorList {
				t.Fatal("validator called after conversion failed")
				return nil
			})
			if err == nil || len(errs) != 0 {
				t.Fatalf("expected a conversion error and no field violations, got %v, %v", err, errs)
			}
			if !strings.HasPrefix(err.Error(), test.errorPrefix) || errors.Unwrap(err) == nil {
				t.Errorf("expected an error with prefix %q wrapping the conversion cause, got %v", test.errorPrefix, err)
			}
		})
	}
}
