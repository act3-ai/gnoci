// Package apis defines api schemas.
package apis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScheme(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		scheme := NewScheme()
		assert.NotNil(t, scheme)
	})
}
