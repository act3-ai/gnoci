package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupportedCommand(t *testing.T) {
	t.Run("Capabilities", func(t *testing.T) {
		ok := SupportedCommand(Capabilities)
		assert.True(t, ok)
	})

	t.Run("Options", func(t *testing.T) {
		ok := SupportedCommand(Options)
		assert.True(t, ok)
	})

	t.Run("List", func(t *testing.T) {
		ok := SupportedCommand(List)
		assert.True(t, ok)
	})

	t.Run("Push", func(t *testing.T) {
		ok := SupportedCommand(Push)
		assert.True(t, ok)
	})

	t.Run("Fetch", func(t *testing.T) {
		ok := SupportedCommand(Fetch)
		assert.True(t, ok)
	})

	// test for real git remote helper commands, forcing an updated unit test
	// if we support them in the future

	t.Run("Unsupported - Import", func(t *testing.T) {
		ok := SupportedCommand(Command("import"))
		assert.False(t, ok)
	})

	t.Run("Unsupported - Export", func(t *testing.T) {
		ok := SupportedCommand(Command("export"))
		assert.False(t, ok)
	})

	t.Run("Unsupported - Connect", func(t *testing.T) {
		ok := SupportedCommand(Command("connect"))
		assert.False(t, ok)
	})

	t.Run("Unsupported - Stateless-Connect", func(t *testing.T) {
		ok := SupportedCommand(Command("stateless-connect"))
		assert.False(t, ok)
	})

	t.Run("Unsupported - Get", func(t *testing.T) {
		ok := SupportedCommand(Command("get"))
		assert.False(t, ok)
	})

	// a bogus case
	t.Run("Unsupported - Not Real Command", func(t *testing.T) {
		ok := SupportedCommand(Command("foo"))
		assert.False(t, ok)
	})
}
