// Package actions holds actions called by the root git-remote-oci command.
package actions

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/pkg/apis"
	"github.com/stretchr/testify/assert"
)

func TestNewGitLFS(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)
		version := "v1.0.0"
		cfgFiles := []string{"/tmp/foo"}

		gotGitLFS := NewGitLFS(in, out, version, cfgFiles)
		assert.Equal(t, version, gotGitLFS.version)
		assert.NotNil(t, gotGitLFS.apiScheme)
		assert.Equal(t, cfgFiles, gotGitLFS.ConfigFiles)
		assert.NotNil(t, gotGitLFS.comm)
	})
}

func TestGitLFS_GetScheme(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		action := &GitLFS{
			apiScheme: apis.NewScheme(),
		}

		action.GetScheme()
	})
}

func TestGitLFS_GetConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		cfgContents := "# Git o(n) OCI Configuration\napiVersion: gnoci.act3-ai.io/v1alpha1\nkind: Configuration\nregistryConfig:\n  registries:\n    127.0.0.1:5000:\n      plainHTTP: true\n"

		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(cfgContents), 0666)
		assert.NoError(t, err)

		action := &GitLFS{
			apiScheme: apis.NewScheme(),
		}

		cfg, err := action.GetConfig(t.Context())
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})
}

func Test_trimProtocol(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := "oci://foobar"
		got := trimProtocol(in)
		assert.False(t, strings.Contains(got, "oci://"))
	})
}
