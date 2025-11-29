// Package v1alpha1 defines the v1alpha1 schema.
//
// +kubebuilder:object:generate=true
package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigurationDefault(t *testing.T) {
	t.Run("Succss", func(t *testing.T) {
		const testReg = "127.0.0.1:5000"

		expected := &Configuration{
			TypeMeta: v1.TypeMeta{
				Kind:       "Configuration",
				APIVersion: GroupVersion.String(),
			},
			ConfigurationSpec: ConfigurationSpec{
				RegistryConfig: RegistryConfig{
					Registries: map[string]Registry{
						testReg: {
							PlainHTTP:    true,
							NonCompliant: false,
						},
					},
				},
			},
		}

		in := &Configuration{
			TypeMeta: v1.TypeMeta{
				Kind:       expected.Kind,
				APIVersion: expected.APIVersion,
			},
			ConfigurationSpec: ConfigurationSpec{
				RegistryConfig: RegistryConfig{
					Registries: map[string]Registry{
						testReg: expected.RegistryConfig.Registries[testReg],
					},
				},
			},
		}

		ConfigurationDefault(in)

		assert.NotNil(t, in)
		assert.True(t, reflect.DeepEqual(expected, in))
	})
}
