// Package actions holds actions called by the root git-remote-oci command.
package actions

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/internal/mocks/modelmock"
	"github.com/act3-ai/gnoci/internal/testutils"
	"github.com/act3-ai/gnoci/pkg/apis"
	"github.com/act3-ai/gnoci/pkg/apis/gnoci.act3-ai.io/v1alpha1"
	"github.com/act3-ai/gnoci/pkg/oci"
	"github.com/act3-ai/gnoci/pkg/protocol/git/comms"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewGit(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)
		gitDir := ".git"
		shortname := "foo"
		address := "oci://example.com/repo/test:sync"
		version := "v1.0.0"
		cfgFiles := []string{"/tmp/foo"}

		gotGit := NewGit(in, out, gitDir, shortname, address, version, cfgFiles)
		assert.NotNil(t, gotGit)
		assert.Equal(t, version, gotGit.version)
		assert.NotNil(t, gotGit.apiScheme)
		assert.Equal(t, cfgFiles, gotGit.ConfigFiles)
		assert.Equal(t, gitDir, gotGit.gitDir)
		assert.Equal(t, shortname, gotGit.name)
		assert.NotNil(t, gotGit.comm)
		assert.False(t, strings.Contains(gotGit.address, "oci://"))
	})
}

func TestGit_localRepo(t *testing.T) {
	t.Run("Not Cached", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := gogit.PlainInit(tmpDir, false)
		assert.NoError(t, err)

		action := &Git{
			local:  nil, // being explicit
			gitDir: tmpDir,
		}

		repo, err := action.localRepo(t.Context())
		assert.NoError(t, err)
		assert.NotNil(t, repo)
	})

	t.Run("Empty Git Dir", func(t *testing.T) {
		action := &Git{
			local:  nil, // being explicit
			gitDir: "",
		}

		repo, err := action.localRepo(t.Context())
		assert.Error(t, err)
		assert.Nil(t, repo)
	})

	t.Run("Not a Git Repository", func(t *testing.T) {
		action := &Git{
			local:  nil, // being explicit
			gitDir: t.TempDir(),
		}

		repo, err := action.localRepo(t.Context())
		assert.Error(t, err)
		assert.Nil(t, repo)
	})
}

func TestGit_handleList(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelMock := modelmock.NewMockModeler(ctrl)
		tmpDir := t.TempDir()

		_, err := gogit.PlainInit(tmpDir, false)
		assert.NoError(t, err)

		expectedHeads := map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.ReferenceName("refs/heads/main"): {
				Commit: "32396c14a264a71cbd47cc7a8678cebb2cdd15ed",
				Layer:  digest.Digest("sha256:eba70958398124d1699b1d5733b916677c9bc2f7629153191eed4d7086976070"),
			},
		}

		expectedTags := map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.ReferenceName("refs/tags/v1.0.0"): {
				Commit: "32396c14a264a71cbd47cc7a8678cebb2cdd15ed",
				Layer:  digest.Digest("sha256:eba70958398124d1699b1d5733b916677c9bc2f7629153191eed4d7086976070"),
			},
		}

		modelMock.EXPECT().
			FetchOrDefault(gomock.Any()).
			Return(ocispec.Descriptor{}, nil)

		modelMock.EXPECT().
			HeadRefs().
			Return(expectedHeads).
			Times(1)

		modelMock.EXPECT().
			TagRefs().
			Return(expectedTags).
			Times(1)

		in := new(bytes.Buffer)
		out := new(bytes.Buffer)

		comm := comms.NewCommunicator(in, out)
		revcomm := testutils.NewReverseCommunicator(out, in)

		err = revcomm.SendListRequest(false)
		assert.NoError(t, err)

		action := &Git{
			local:  nil, // being explicit
			gitDir: tmpDir,
			remote: modelMock,
			comm:   comm,
		}

		err = action.handleList(t.Context())
		assert.NoError(t, err)
	})
}

func TestGit_GetScheme(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		action := &Git{
			apiScheme: apis.NewScheme(),
		}

		action.GetScheme()
	})
}

func TestGit_GetConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		cfgContents := "# Git o(n) OCI Configuration\napiVersion: gnoci.act3-ai.io/v1alpha1\nkind: Configuration\nregistryConfig:\n  registries:\n    127.0.0.1:5000:\n      plainHTTP: true\n"

		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(cfgContents), 0666)
		assert.NoError(t, err)

		action := &Git{
			apiScheme: apis.NewScheme(),
		}

		cfg, err := action.GetConfig(t.Context())
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})
}

func Test_repoOptsFromConfig(t *testing.T) {
	t.Run("Plain HTTP Enabled", func(t *testing.T) {
		host := "example.com"
		cfg := v1alpha1.Configuration{
			ConfigurationSpec: v1alpha1.ConfigurationSpec{
				RegistryConfig: v1alpha1.RegistryConfig{
					Registries: map[string]v1alpha1.Registry{
						host: {
							PlainHTTP: true,
						},
					},
				},
			},
		}

		gotOpts := repoOptsFromConfig(host, &cfg)
		assert.NotNil(t, gotOpts)

		assert.True(t, gotOpts.PlainHTTP)
	})

	t.Run("NonCompliant Enabled", func(t *testing.T) {
		host := "example.com"
		cfg := v1alpha1.Configuration{
			ConfigurationSpec: v1alpha1.ConfigurationSpec{
				RegistryConfig: v1alpha1.RegistryConfig{
					Registries: map[string]v1alpha1.Registry{
						host: {
							NonCompliant: true,
						},
					},
				},
			},
		}

		gotOpts := repoOptsFromConfig(host, &cfg)
		assert.NotNil(t, gotOpts)

		assert.True(t, gotOpts.NonCompliant)
	})
}
