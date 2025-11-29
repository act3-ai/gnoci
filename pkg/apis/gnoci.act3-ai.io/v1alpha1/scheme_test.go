// Package v1alpha1 defines the v1alpha1 schema.
//
// +kubebuilder:object:generate=true
package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_addKnownTypes(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		scheme := runtime.NewScheme()
		err := addKnownTypes(scheme)
		assert.NoError(t, err)
	})
}
