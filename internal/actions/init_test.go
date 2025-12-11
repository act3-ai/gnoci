package actions

import (
	"os"
	"testing"

	"github.com/act3-ai/gnoci/internal/ociutil"
	"github.com/stretchr/testify/assert"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

var testRemote = registry.Reference{
	Registry:   "reg.example.com",
	Repository: "repo",
	Reference:  "tag",
}

func Test_initRemoteConn(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// more robust test in ociutil, here we mainly care about fstorePath and fstore
		expectedOpts := ociutil.RepositoryOptions{
			UserAgent:     "foo",
			PlainHTTP:     true,
			NonCompliant:  true,
			RegistryCreds: credentials.NewMemoryStore(),
		}

		gt, fstorePath, fstore, err := initRemoteConn(t.Context(), testRemote, &expectedOpts)
		assert.NoError(t, err)
		assert.False(t, fstorePath == "")
		defer func() {
			err = os.RemoveAll(fstorePath)
			assert.NoError(t, err)
		}()
		assert.NotNil(t, fstore)
		assert.NotNil(t, gt)
	})
}
