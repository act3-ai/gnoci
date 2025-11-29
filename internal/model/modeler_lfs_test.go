package model

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/act3-ai/gnoci/internal/progress"
	"github.com/act3-ai/gnoci/pkg/oci"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/memory"
)

// returns git manifest, git config, lfs manifest
func setupRemoteWithLFS(t *testing.T, gt oras.GraphTarget) (ocispec.Manifest, oci.ConfigGit, ocispec.Manifest) {
	// We intentionally don't use [oras.PackManifest] in case of upstream bugs,
	// although they're quite stable so perhaps we're unnecessarily cautious
	t.Helper()

	gitMan, gitConfig := setupRemote(t, gt)
	gitManDesc, err := gt.Resolve(t.Context(), testRemote.String())
	assert.NoError(t, err)

	pushLFSConfig(t, gt)
	lfsLayerDesc := pushLFSLayer(t, gt)
	lfsManifest := pushLFSManifest(t, gt, &gitManDesc, []ocispec.Descriptor{lfsLayerDesc})

	return gitMan, gitConfig, lfsManifest

}

func pushLFSConfig(t *testing.T, gt oras.GraphTarget) {
	t.Helper()

	err := gt.Push(t.Context(), ocispec.DescriptorEmptyJSON, bytes.NewReader(ocispec.DescriptorEmptyJSON.Data))
	assert.NoError(t, err)
}

func pushLFSLayer(t *testing.T, gt oras.GraphTarget) ocispec.Descriptor {
	t.Helper()

	// https://en.wikipedia.org/wiki/Gnocchi
	lfsLayer := []byte("Gnocchi are commonly cooked in salted boiling water and then dressed with various sauces.They are usually eaten as a first course (primo) as an alternative to soups (minestre) or pasta, but they can also be served as a contorno (side dish) to some main courses.")
	lfsLayerDgst := digest.FromBytes(lfsLayer)

	lfsLayerDesc := ocispec.Descriptor{
		MediaType: oci.MediaTypeLFSLayer,
		Digest:    lfsLayerDgst,
		Size:      int64(len(lfsLayer)),
	}

	err := gt.Push(t.Context(), lfsLayerDesc, bytes.NewReader(lfsLayer))
	assert.NoError(t, err)
	return lfsLayerDesc
}

func pushLFSManifest(t *testing.T, gt oras.GraphTarget, subject *ocispec.Descriptor, layers []ocispec.Descriptor) ocispec.Manifest {
	t.Helper()

	// lfsManifest metadata
	lfsManifest := ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: oci.ArtifactTypeLFSManifest,
		Config:       ocispec.DescriptorEmptyJSON,
		Layers:       layers,
		Annotations: map[string]string{
			ocispec.AnnotationCreated: "1970-01-01T00:00:00Z",
		},
		Subject: subject,
	}

	lfsManifestRaw, err := json.Marshal(lfsManifest)
	assert.NoError(t, err)

	lfsManifestDesc := ocispec.Descriptor{
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: oci.ArtifactTypeLFSManifest,
		Digest:       digest.FromBytes(lfsManifestRaw),
		Size:         int64(len(lfsManifestRaw)),
		Annotations: map[string]string{
			ocispec.AnnotationCreated: "1970-01-01T00:00:00Z",
		},
	}

	err = gt.Push(t.Context(), lfsManifestDesc, bytes.NewReader(lfsManifestRaw))
	assert.NoError(t, err)

	return lfsManifest
}

func TestNewLFSModeler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		fstore, err := file.New(t.TempDir())
		assert.NoError(t, err)
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		gt := memory.New()

		lfsModeler := NewLFSModeler(testRemote, fstore, gt)
		assert.NotNil(t, lfsModeler)

		model, ok := lfsModeler.(*model)
		assert.True(t, ok)

		assert.Equal(t, testRemote, model.ref)
		assert.Equal(t, fstore, model.fstore)
		assert.Equal(t, gt, model.gt)
	})
}

func Test_model_FetchLFS(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		gt := memory.New()
		gitManifest, gitConfig, lfsManifest := setupRemoteWithLFS(t, gt)

		fstore, err := file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		m := &model{
			ref:         testRemote,
			gt:          gt,
			fstore:      fstore,
			fetched:     true,
			manDesc:     *lfsManifest.Subject,
			man:         gitManifest,
			cfg:         gitConfig,
			refsByLayer: map[digest.Digest][]plumbing.Hash{},
			newPacks:    nil,
		}

		_, err = m.FetchLFS(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, lfsManifest, m.lfsMan)

		err = fstore.Close()
		assert.NoError(t, err)
	})

	t.Run("No Referrers", func(t *testing.T) {
		gt := memory.New()
		gitManifest, gitConfig, _ := setupRemoteWithLFS(t, gt)

		fstore, err := file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		m := &model{
			ref:     testRemote,
			gt:      gt,
			fstore:  fstore,
			fetched: true,
			// manDesc:     *lfsManifest.Subject,
			man:         gitManifest,
			cfg:         gitConfig,
			refsByLayer: map[digest.Digest][]plumbing.Hash{},
			newPacks:    nil,
		}

		_, err = m.FetchLFS(t.Context())
		assert.Error(t, err)
	})
}

func Test_model_FetchLFSOrDefault(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		gt := memory.New()
		gitManifest, gitConfig, lfsManifest := setupRemoteWithLFS(t, gt)

		fstore, err := file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		m := &model{
			ref:         testRemote,
			gt:          gt,
			fstore:      fstore,
			fetched:     true,
			manDesc:     *lfsManifest.Subject,
			man:         gitManifest,
			cfg:         gitConfig,
			refsByLayer: map[digest.Digest][]plumbing.Hash{},
			newPacks:    nil,
		}

		_, err = m.FetchLFSOrDefault(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, lfsManifest, m.lfsMan)

		err = fstore.Close()
		assert.NoError(t, err)
	})

	t.Run("Defaulted", func(t *testing.T) {
		// setup remote without LFS referrer
		gt := memory.New()
		gitManifest, gitConfig := setupRemote(t, gt)

		fstore, err := file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		m := &model{
			ref:     testRemote,
			gt:      gt,
			fstore:  fstore,
			fetched: true,
			// manDesc:     *lfsManifest.Subject,
			man:         gitManifest,
			cfg:         gitConfig,
			refsByLayer: map[digest.Digest][]plumbing.Hash{},
			newPacks:    nil,
		}

		_, err = m.FetchLFSOrDefault(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, ocispec.Manifest{}, m.lfsMan)

		err = fstore.Close()
		assert.NoError(t, err)
	})
}

func Test_model_FetchLFSLayer(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		gt := memory.New()
		gitManifest, gitConfig, lfsManifest := setupRemoteWithLFS(t, gt)

		fstore, err := file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		m := &model{
			ref:         testRemote,
			gt:          gt,
			fstore:      fstore,
			fetched:     true,
			manDesc:     *lfsManifest.Subject,
			man:         gitManifest,
			cfg:         gitConfig,
			refsByLayer: map[digest.Digest][]plumbing.Hash{},
			newPacks:    nil,
		}

		_, err = m.FetchLFS(t.Context())
		assert.NoError(t, err)
		assert.Equal(t, lfsManifest, m.lfsMan)

		rc, err := m.FetchLFSLayer(t.Context(), lfsManifest.Layers[0].Digest, &FetchLFSOptions{})
		assert.NoError(t, err)
		defer rc.Close()

		err = fstore.Close()
		assert.NoError(t, err)
	})
}

func Test_model_PushLFSManifest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// setup remote without LFS referrer
		gt := memory.New()
		gitManifest, gitConfig := setupRemote(t, gt)

		fstore, err := file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		m := &model{
			ref:         testRemote,
			gt:          gt,
			fstore:      fstore,
			fetched:     true,
			man:         gitManifest,
			cfg:         gitConfig,
			refsByLayer: map[digest.Digest][]plumbing.Hash{},
			newPacks:    nil,
			lfsMan:      ocispec.Manifest{},
		}

		lfsFilePath := filepath.Join(t.TempDir(), "foolfs")
		f, err := os.OpenFile(lfsFilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		assert.NoError(t, err)
		defer f.Close()

		lfsFileContents := "example file contents"
		_, err = f.WriteString(lfsFileContents)
		assert.NoError(t, err)
		err = f.Close()
		assert.NoError(t, err)

		expectedLFSDesc := ocispec.Descriptor{
			MediaType: oci.MediaTypeLFSLayer,
			Digest:    digest.FromString(lfsFileContents),
			Size:      int64(len(lfsFileContents)),
			Annotations: map[string]string{
				ocispec.AnnotationTitle: filepath.Base(lfsFilePath),
			},
		}

		lfsDesc, err := m.PushLFSFile(t.Context(), lfsFilePath, &PushLFSOptions{})
		assert.NoError(t, err)
		assert.Equal(t, expectedLFSDesc, lfsDesc)

		gitManDesc, err := gt.Resolve(t.Context(), testRemote.String())
		assert.NoError(t, err)

		lfsManDesc, err := m.PushLFSManifest(t.Context(), gitManDesc)
		assert.NoError(t, err)

		lfsManRaw, err := content.FetchAll(t.Context(), gt, lfsManDesc)
		assert.NoError(t, err)

		var lfsMan ocispec.Manifest
		err = json.Unmarshal(lfsManRaw, &lfsMan)
		assert.NoError(t, err)

		assert.NotNil(t, lfsMan.Layers)
		assert.Equal(t, 1, len(lfsMan.Layers))
		assert.Equal(t, expectedLFSDesc, lfsMan.Layers[0])

		err = fstore.Close()
		assert.NoError(t, err)
	})
}

func Test_model_PushLFSFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// setup remote without LFS referrer
		gt := memory.New()
		gitManifest, gitConfig := setupRemote(t, gt)

		fstore, err := file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		m := &model{
			ref:         testRemote,
			gt:          gt,
			fstore:      fstore,
			fetched:     true,
			man:         gitManifest,
			cfg:         gitConfig,
			refsByLayer: map[digest.Digest][]plumbing.Hash{},
			newPacks:    nil,
			lfsMan:      ocispec.Manifest{},
		}

		lfsFilePath := filepath.Join(t.TempDir(), "foolfs")
		f, err := os.OpenFile(lfsFilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		assert.NoError(t, err)
		defer f.Close()

		lfsFileContents := "example file contents"
		_, err = f.WriteString(lfsFileContents)
		assert.NoError(t, err)
		err = f.Close()
		assert.NoError(t, err)

		expectedLFSDesc := ocispec.Descriptor{
			MediaType: oci.MediaTypeLFSLayer,
			Digest:    digest.FromString(lfsFileContents),
			Size:      int64(len(lfsFileContents)),
			Annotations: map[string]string{
				ocispec.AnnotationTitle: filepath.Base(lfsFilePath),
			},
		}

		lfsDesc, err := m.PushLFSFile(t.Context(), lfsFilePath, &PushLFSOptions{})
		assert.NoError(t, err)
		assert.Equal(t, expectedLFSDesc, lfsDesc)

		lfsFileRaw, err := content.FetchAll(t.Context(), gt, lfsDesc)
		assert.NoError(t, err)
		assert.Equal(t, lfsFileContents, string(lfsFileRaw))

		err = fstore.Close()
		assert.NoError(t, err)
	})

	t.Run("Deduplicate", func(t *testing.T) {
		// setup remote without LFS referrer
		gt := memory.New()
		gitManifest, gitConfig := setupRemote(t, gt)

		fstore, err := file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()

		m := &model{
			ref:         testRemote,
			gt:          gt,
			fstore:      fstore,
			fetched:     true,
			man:         gitManifest,
			cfg:         gitConfig,
			refsByLayer: map[digest.Digest][]plumbing.Hash{},
			newPacks:    nil,
			lfsMan:      ocispec.Manifest{},
		}

		lfsFilePath := filepath.Join(t.TempDir(), "foolfs")
		f, err := os.OpenFile(lfsFilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		assert.NoError(t, err)
		defer f.Close()

		lfsFileContents := "example file contents"
		_, err = f.WriteString(lfsFileContents)
		assert.NoError(t, err)
		err = f.Close()
		assert.NoError(t, err)

		expectedLFSDesc := ocispec.Descriptor{
			MediaType: oci.MediaTypeLFSLayer,
			Digest:    digest.FromString(lfsFileContents),
			Size:      int64(len(lfsFileContents)),
			Annotations: map[string]string{
				ocispec.AnnotationTitle: filepath.Base(lfsFilePath),
			},
		}

		lfsDesc, err := m.PushLFSFile(t.Context(), lfsFilePath, &PushLFSOptions{})
		assert.NoError(t, err)
		assert.Equal(t, expectedLFSDesc, lfsDesc)

		lfsFileRaw, err := content.FetchAll(t.Context(), gt, lfsDesc)
		assert.NoError(t, err)
		assert.Equal(t, lfsFileContents, string(lfsFileRaw))

		// create a new filestore to avoid it's complaints
		err = fstore.Close()
		assert.NoError(t, err)
		fstore, err = file.New(t.TempDir())
		if err != nil {
			t.Errorf("initializing OCI filestore: %v", err)
		}
		defer func() {
			if err := fstore.Close(); err != nil {
				t.Errorf("closing OCI filestore: %v", err)
			}
		}()
		m.fstore = fstore

		// again, as a duplicate
		lfsDesc, err = m.PushLFSFile(t.Context(), lfsFilePath, &PushLFSOptions{})
		assert.NoError(t, err)
		assert.Equal(t, expectedLFSDesc, lfsDesc)
		assert.Equal(t, 1, len(m.lfsMan.Layers))

	})
}

func Test_progressOrDefault(t *testing.T) {
	t.Run("Progress Enabled", func(t *testing.T) {
		ch := make(chan progress.Progress)
		opts := ProgressOptions{
			Info: ch,
		}

		rc := io.NopCloser(strings.NewReader("foo"))

		gotRC := progressOrDefault(t.Context(), &opts, rc)
		assert.NotNil(t, gotRC)

		_, ok := gotRC.(progress.EvalReadCloser)
		assert.True(t, ok)
	})

	t.Run("Progress Disabled", func(t *testing.T) {
		opts := ProgressOptions{}

		rc := io.NopCloser(strings.NewReader("foo"))

		gotRC := progressOrDefault(t.Context(), &opts, rc)
		assert.NotNil(t, gotRC)

		_, ok := gotRC.(progress.EvalReadCloser)
		assert.False(t, ok)
	})
}
