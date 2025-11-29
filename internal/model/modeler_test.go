// Package model provides utility functions for modeling a git repository in OCI.
package model

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"

	"github.com/act3-ai/gnoci/pkg/oci"
)

var testRemote = registry.Reference{
	Registry:   "reg.example.com",
	Repository: "repo",
	Reference:  "tag",
}

// setupRemote pushes a Git OCI artifact, returning anything needed for validation
func setupRemote(t *testing.T, gt oras.GraphTarget) (ocispec.Manifest, oci.ConfigGit) {
	// We intentionally don't use [oras.PackManifest] in case of upstream bugs,
	// although they're quite stable so perhaps we're unnecessarily cautious
	t.Helper()

	// https://en.wikipedia.org/wiki/Gnocchi
	layer := []byte("Gnocchi are a varied family of pasta-like dumplings in Italian cuisine.")
	layerDgst := digest.FromBytes(layer)

	layerDesc := ocispec.Descriptor{
		MediaType: oci.MediaTypePackLayer,
		Digest:    layerDgst,
		Size:      int64(len(layer)),
	}

	// config metadata
	config := oci.ConfigGit{
		Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.Main: {
				Commit: plumbing.ZeroHash.String(),
				Layer:  layerDgst,
			},
		},
		Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{
			"refs/tags/foobar": {
				Commit: plumbing.ZeroHash.String(),
				Layer:  layerDgst,
			},
		},
	}

	configRaw, err := json.Marshal(config)
	assert.NoError(t, err)

	configDesc := ocispec.Descriptor{
		MediaType: oci.MediaTypeGitConfig,
		Digest:    digest.FromBytes(configRaw),
		Size:      int64(len(configRaw)),
	}

	// manifest metadata
	manifest := ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: oci.ArtifactTypeGitManifest,
		Config:       configDesc,
		Layers:       []ocispec.Descriptor{layerDesc}, // WARNING: changes will break tests
		Annotations: map[string]string{
			ocispec.AnnotationCreated: "1970-01-01T00:00:00Z",
		},
	}

	manifestRaw, err := json.Marshal(manifest)
	assert.NoError(t, err)

	manifestDesc := ocispec.Descriptor{
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: oci.ArtifactTypeGitManifest,
		Digest:       digest.FromBytes(manifestRaw),
		Size:         int64(len(manifestRaw)),
		Annotations: map[string]string{
			ocispec.AnnotationCreated: "1970-01-01T00:00:00Z",
		},
	}

	err = gt.Push(t.Context(), layerDesc, bytes.NewReader(layer))
	assert.NoError(t, err)

	err = gt.Push(t.Context(), configDesc, bytes.NewReader(configRaw))
	assert.NoError(t, err)

	err = gt.Push(t.Context(), manifestDesc, bytes.NewReader(manifestRaw))
	assert.NoError(t, err)

	err = gt.Tag(t.Context(), manifestDesc, testRemote.String())
	assert.NoError(t, err)

	return manifest, config
}

func TestNewModeler(t *testing.T) {
	gt := memory.New()

	fstore, err := file.New(t.TempDir())
	assert.NoError(t, err)
	defer func() {
		if err := fstore.Close(); err != nil {
			t.Errorf("closing OCI filestore: %v", err)
		}
	}()

	got := NewModeler(testRemote, fstore, gt)

	model := got.(*model)

	assert.Equal(t, testRemote, model.ref)
	assert.Equal(t, fstore, model.fstore)
	assert.Equal(t, gt, model.gt)
}

func Test_model_Fetch(t *testing.T) {
	// sharing a remote between tests is safe as long as we only fetch from it.
	// TODO: Consider mocking oras.GraphTarget. Note that at some level we're just
	// testing their interface but it could have value in ensuring we aren't
	// fetching things more than once, but perhaps the fetched boolean is sufficient...
	gt := memory.New()
	manifest, config := setupRemote(t, gt)

	expectedRefsByLayer := map[digest.Digest][]plumbing.Hash{
		manifest.Layers[0].Digest: {plumbing.ZeroHash},
	}

	type fields struct {
		fetched bool
	}

	tests := []struct {
		name   string
		fields fields
		remote registry.Reference
		wantFn func(t *testing.T, m *model, err error)
	}{
		{
			name:   "Success",
			fields: fields{fetched: false},
			remote: testRemote,
			wantFn: func(t *testing.T, m *model, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.True(t, m.fetched)
				assert.Equal(t, manifest, m.man)
				assert.Equal(t, config, m.cfg)
				assert.Nil(t, m.newPacks)
				assert.Equal(t, expectedRefsByLayer, m.refsByLayer)
			},
		},
		{
			name:   "Already Fetched",
			fields: fields{fetched: true},
			remote: testRemote,
			wantFn: func(t *testing.T, m *model, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.True(t, m.fetched)
				assert.Nil(t, m.newPacks)
			},
		},
		{
			name:   "No Existing Manifest",
			fields: fields{fetched: false},
			remote: registry.Reference{
				Registry:   "reg.dne",
				Repository: "doesnotexist",
				Reference:  "tag",
			},
			wantFn: func(t *testing.T, m *model, err error) {
				t.Helper()

				assert.Error(t, err)
				assert.False(t, m.fetched)
				assert.Nil(t, m.newPacks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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
				ref:         tt.remote,
				gt:          gt,
				fstore:      fstore,
				fetched:     tt.fields.fetched,
				man:         ocispec.Manifest{},
				cfg:         oci.ConfigGit{},
				refsByLayer: map[digest.Digest][]plumbing.Hash{},
				newPacks:    nil,
			}

			_, err = m.Fetch(t.Context())
			tt.wantFn(t, m, err)

			err = fstore.Close()
			assert.NoError(t, err)
		})
	}
}

func Test_model_FetchOrDefault(t *testing.T) {
	// sharing a remote between tests is safe as long as we only fetch from it.
	gt := memory.New()
	manifest, config := setupRemote(t, gt)

	expectedRefsByLayer := map[digest.Digest][]plumbing.Hash{
		manifest.Layers[0].Digest: {plumbing.ZeroHash},
	}

	type fields struct {
		fetched bool
	}

	tests := []struct {
		name   string
		fields fields
		remote registry.Reference
		wantFn func(t *testing.T, m *model, err error)
	}{
		{
			name:   "Success",
			fields: fields{fetched: false},
			remote: testRemote,
			wantFn: func(t *testing.T, m *model, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.True(t, m.fetched)
				assert.Equal(t, manifest, m.man)
				assert.Equal(t, config, m.cfg)
				assert.Nil(t, m.newPacks)
				assert.Equal(t, expectedRefsByLayer, m.refsByLayer)
			},
		},
		{
			name:   "Already Fetched",
			fields: fields{fetched: true},
			remote: testRemote,
			wantFn: func(t *testing.T, m *model, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.True(t, m.fetched)
				assert.Nil(t, m.newPacks)
			},
		},
		{
			name:   "Defaulted",
			fields: fields{fetched: false},
			remote: registry.Reference{
				Registry:   "reg.dne",
				Repository: "doesnotexist",
				Reference:  "tag",
			},
			wantFn: func(t *testing.T, m *model, err error) {
				t.Helper()

				expectedEmptyManifest := ocispec.Manifest{
					MediaType:    ocispec.MediaTypeImageManifest,
					ArtifactType: oci.ArtifactTypeGitManifest,
				}

				expectedEmptyConfig := oci.ConfigGit{
					Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
						tempGitManifest: {
							Commit: "foo",
						},
					},
					Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{},
				}

				expectedRefsByLayer := map[digest.Digest][]plumbing.Hash{}

				assert.NoError(t, err)
				assert.True(t, m.fetched)
				assert.Equal(t, expectedEmptyManifest, m.man)
				assert.Equal(t, len(expectedEmptyConfig.Heads), len(m.cfg.Heads))
				_, ok := m.cfg.Heads[tempGitManifest]
				assert.True(t, ok)
				assert.Equal(t, expectedEmptyConfig.Tags, m.cfg.Tags)
				assert.Equal(t, expectedRefsByLayer, m.refsByLayer)
				assert.Nil(t, m.newPacks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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
				ref:         tt.remote,
				gt:          gt,
				fstore:      fstore,
				fetched:     tt.fields.fetched,
				man:         ocispec.Manifest{},
				cfg:         oci.ConfigGit{},
				refsByLayer: map[digest.Digest][]plumbing.Hash{},
				newPacks:    nil,
			}

			_, err = m.FetchOrDefault(t.Context())
			tt.wantFn(t, m, err)

			err = fstore.Close()
			assert.NoError(t, err)
		})
	}
}

func Test_model_FetchLayer(t *testing.T) {
	// sharing a remote between tests is safe as long as we only fetch from it.
	gt := memory.New()
	manifest, _ := setupRemote(t, gt)

	type fields struct {
		fetched bool
	}

	tests := []struct {
		name   string
		fields fields
		remote registry.Reference
		wantFn func(t *testing.T, m *model, err error)
	}{
		{
			name:   "Success",
			fields: fields{fetched: false},
			remote: testRemote,
			wantFn: func(t *testing.T, m *model, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.Nil(t, m.newPacks)
			},
		},
		{
			name:   "Layer Not in Manifest",
			fields: fields{fetched: true},
			remote: testRemote,
			wantFn: func(t *testing.T, m *model, err error) {
				t.Helper()

				assert.ErrorIs(t, err, errLayerNotInManifest)
				assert.Nil(t, m.newPacks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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
				ref:         tt.remote,
				gt:          gt,
				fstore:      fstore,
				fetched:     tt.fields.fetched,
				man:         ocispec.Manifest{},
				cfg:         oci.ConfigGit{},
				refsByLayer: map[digest.Digest][]plumbing.Hash{},
				newPacks:    nil,
			}

			// fetching metadata is a prerequisite to fetching layers
			_, err = m.Fetch(t.Context())
			assert.NoError(t, err)

			rc, err := m.FetchLayer(t.Context(), manifest.Layers[0].Digest)

			tt.wantFn(t, m, err)
			if rc != nil {
				err = rc.Close()
				assert.NoError(t, err)
			}

			err = fstore.Close()
			assert.NoError(t, err)
		})
	}
}

func Test_model_Push(t *testing.T) {
	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "layer-file-*.pack")
	assert.NoError(t, err)

	layerContents := "Gnocchi are a varied family of pasta-like dumplings in Italian cuisine."
	expectedLayerDesc := ocispec.Descriptor{
		MediaType: oci.MediaTypePackLayer,
		Digest:    digest.FromString(layerContents),
		Size:      int64(len(layerContents)),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: path.Base(f.Name()),
		},
	}
	_, err = f.WriteString(layerContents)
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)

	expectedConfig := oci.ConfigGit{
		Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.ReferenceName("refs/heads/main"): {Commit: plumbing.ZeroHash.String(), Layer: expectedLayerDesc.Digest},
			plumbing.ReferenceName("refs/heads/foo"):  {Commit: plumbing.ZeroHash.String(), Layer: expectedLayerDesc.Digest},
		},
		Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{
			plumbing.ReferenceName("refs/tags/foo"): {Commit: plumbing.ZeroHash.String(), Layer: expectedLayerDesc.Digest},
			plumbing.ReferenceName("refs/tags/bar"): {Commit: plumbing.ZeroHash.String(), Layer: expectedLayerDesc.Digest},
		},
	}

	expectedConfigRaw, err := json.Marshal(expectedConfig)
	assert.NoError(t, err)

	expectedConfigDesc := ocispec.Descriptor{
		MediaType: oci.MediaTypeGitConfig,
		Digest:    digest.FromBytes(expectedConfigRaw),
		Size:      int64(len(expectedConfigRaw)),
	}

	expectedManifest := ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: oci.ArtifactTypeGitManifest,
		Config:       expectedConfigDesc,
		Layers:       []ocispec.Descriptor{expectedLayerDesc}, // WARNING: changes will break tests
		Annotations: map[string]string{
			ocispec.AnnotationCreated: "1970-01-01T00:00:00Z",
		},
	}

	expectedManifestRaw, err := json.Marshal(expectedManifest)
	assert.NoError(t, err)

	expectedManDesc := ocispec.Descriptor{
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: oci.ArtifactTypeGitManifest,
		Digest:       digest.FromBytes(expectedManifestRaw),
		Size:         int64(len(expectedManifestRaw)),
		Annotations: map[string]string{
			ocispec.AnnotationCreated: "1970-01-01T00:00:00Z",
		},
	}

	tests := []struct {
		name     string
		newPacks []ocispec.Descriptor
		wantFn   func(t *testing.T, m *model, manDesc ocispec.Descriptor, gt oras.GraphTarget, err error)
	}{
		{
			name:     "New Packs",
			newPacks: []ocispec.Descriptor{expectedLayerDesc},
			wantFn: func(t *testing.T, m *model, manDesc ocispec.Descriptor, gt oras.GraphTarget, err error) {
				t.Helper()

				// validate manifest push
				assert.NoError(t, err)
				assert.Equal(t, expectedManDesc, manDesc)

				// validate newPacks handling
				assert.Equal(t, []ocispec.Descriptor{expectedLayerDesc}, m.newPacks)

				// validate tag
				gotManDesc, err := gt.Resolve(t.Context(), testRemote.String())
				assert.NoError(t, err)
				assert.Equal(t, expectedManDesc, gotManDesc)

				// validate config push
				rc, err := gt.Fetch(t.Context(), expectedConfigDesc)
				assert.NoError(t, err)
				gotConfigRaw, err := io.ReadAll(rc)
				assert.NoError(t, err)
				err = rc.Close()
				assert.NoError(t, err)

				var gotConfig oci.ConfigGit
				err = json.Unmarshal(gotConfigRaw, &gotConfig)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, gotConfig)
			},
		},
		{
			name:     "No New Packs",
			newPacks: []ocispec.Descriptor{},
			wantFn: func(t *testing.T, m *model, manDesc ocispec.Descriptor, gt oras.GraphTarget, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, m.cfg)
				assert.Equal(t, []ocispec.Descriptor{}, m.newPacks)

				rc, err := gt.Fetch(t.Context(), expectedLayerDesc)
				assert.ErrorIs(t, err, errdef.ErrNotFound)
				if err == nil {
					rc.Close()
				}

				_, err = gt.Resolve(t.Context(), testRemote.String())
				assert.NoError(t, err)
			},
		},
	}

	fstore, err := file.New(t.TempDir())
	if err != nil {
		t.Errorf("initializing OCI filestore: %v", err)
	}
	defer func() {
		if err := fstore.Close(); err != nil {
			t.Errorf("closing OCI filestore: %v", err)
		}
	}()

	err = fstore.Push(t.Context(), expectedLayerDesc, strings.NewReader(layerContents))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gt := memory.New()

			m := &model{
				ref:    testRemote,
				gt:     gt,
				fstore: fstore,
				man: ocispec.Manifest{
					Layers: []ocispec.Descriptor{expectedLayerDesc},
				},
				cfg:         expectedConfig,
				refsByLayer: map[digest.Digest][]plumbing.Hash{},
				newPacks:    tt.newPacks,
			}

			manDesc, err := m.Push(t.Context(), UpdateLFSReferrer(m))

			tt.wantFn(t, m, manDesc, gt, err)
		})
	}

	err = fstore.Close()
	assert.NoError(t, err)
}

func Test_model_AddPack(t *testing.T) {
	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "layer-file-*.pack")
	assert.NoError(t, err)
	layerPath := f.Name()

	layerContents := "Gnocchi are a varied family of pasta-like dumplings in Italian cuisine."
	expectedLayerDesc := ocispec.Descriptor{
		MediaType: oci.MediaTypePackLayer,
		Digest:    digest.FromString(layerContents),
		Size:      int64(len(layerContents)),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: path.Base(f.Name()),
		},
	}
	_, err = f.WriteString(layerContents)
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)

	tests := []struct {
		name     string
		headRefs []*plumbing.Reference
		tagRefs  []*plumbing.Reference
		wantFn   func(t *testing.T, m *model, packDesc ocispec.Descriptor, err error)
	}{
		{
			name: "Many References",
			headRefs: []*plumbing.Reference{
				plumbing.NewHashReference("refs/heads/main", plumbing.ZeroHash),
				plumbing.NewHashReference("refs/heads/foo", plumbing.ZeroHash),
			},
			tagRefs: []*plumbing.Reference{
				plumbing.NewHashReference("refs/tags/foo", plumbing.ZeroHash),
				plumbing.NewHashReference("refs/tags/bar", plumbing.ZeroHash),
			},
			wantFn: func(t *testing.T, m *model, packDesc ocispec.Descriptor, err error) {
				t.Helper()

				expectedConfig := oci.ConfigGit{
					Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
						plumbing.ReferenceName("refs/heads/main"): {Commit: plumbing.ZeroHash.String(), Layer: expectedLayerDesc.Digest},
						plumbing.ReferenceName("refs/heads/foo"):  {Commit: plumbing.ZeroHash.String(), Layer: expectedLayerDesc.Digest},
					},
					Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{
						plumbing.ReferenceName("refs/tags/foo"): {Commit: plumbing.ZeroHash.String(), Layer: expectedLayerDesc.Digest},
						plumbing.ReferenceName("refs/tags/bar"): {Commit: plumbing.ZeroHash.String(), Layer: expectedLayerDesc.Digest},
					},
				}

				assert.NoError(t, err)
				assert.Equal(t, expectedLayerDesc, packDesc)
				assert.Equal(t, expectedConfig, m.cfg)
				assert.Equal(t, []ocispec.Descriptor{expectedLayerDesc}, m.newPacks)
			},
		},
		{
			name: "No References",
			wantFn: func(t *testing.T, m *model, packDesc ocispec.Descriptor, err error) {
				t.Helper()

				assert.Equal(t, oci.ConfigGit{Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{}, Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{}}, m.cfg)
				assert.Equal(t, []ocispec.Descriptor{expectedLayerDesc}, m.newPacks)
			},
		},
		{
			name: "Unsupported Reference Type",
			headRefs: []*plumbing.Reference{
				plumbing.NewHashReference("refs/notes/foo", plumbing.ZeroHash),
			},
			wantFn: func(t *testing.T, m *model, packDesc ocispec.Descriptor, err error) {
				t.Helper()

				assert.ErrorIs(t, err, ErrUnsupportedReferenceType)
				assert.Equal(t, []ocispec.Descriptor{expectedLayerDesc}, m.newPacks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gt := memory.New()

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
				gt:     gt,
				fstore: fstore,
				man: ocispec.Manifest{
					Layers: []ocispec.Descriptor{},
				},
				cfg: oci.ConfigGit{
					Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{},
					Tags:  map[plumbing.ReferenceName]oci.ReferenceInfo{},
				},
				refsByLayer: map[digest.Digest][]plumbing.Hash{},
				newPacks:    nil,
			}

			packDesc, err := m.AddPack(t.Context(), layerPath, append(tt.headRefs, tt.tagRefs...)...)

			tt.wantFn(t, m, packDesc, err)

			err = fstore.Close()
			assert.NoError(t, err)
		})
	}
}

func Test_model_UpdateRef(t *testing.T) {
	const (
		digestAlpha = digest.Digest("sha256:ffbeaa9e113a29d9fc4f58f821e16f594e332033b277ea829eafab12ba148589")
		digestBeta  = digest.Digest("sha256:60290b69da490356c62dc190efe44ca597ec538f792c2908a8a7ec352dc13e5e")
		digestDNE   = digest.Digest("sha256:84290ebe369d11cd880bfa160ef8cbb7c0fc03b8093895534f30263b78f030b2")

		commitAlpha = "eaba08b8fae96b96fe68d88dd311ffb8ca22ba74"
		commitBeta  = "9f9daae4bb300543116a1508cd9ed87bafd9d5fc"
	)

	var (
		headRefName    = plumbing.NewBranchReferenceName("branchfoo")
		tagRefName     = plumbing.NewTagReferenceName("tagbar")
		unsupportedRef = plumbing.NewNoteReferenceName("notesfoobar")
	)

	t.Run("Success - Branch Ref", func(t *testing.T) {
		m := &model{
			man: ocispec.Manifest{
				Layers: []ocispec.Descriptor{
					{Digest: digestAlpha},
					{Digest: digestBeta},
				},
			},
			cfg: oci.ConfigGit{
				Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
					headRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		err := m.UpdateRef(t.Context(), plumbing.NewHashReference(headRefName, plumbing.NewHash(commitBeta)), digestBeta)

		assert.NoError(t, err)
		assert.Equal(t, m.cfg.Heads[headRefName].Commit, commitBeta)
		assert.Equal(t, m.cfg.Heads[headRefName].Layer, digestBeta)
	})

	t.Run("Success - Tag Ref", func(t *testing.T) {
		m := &model{
			man: ocispec.Manifest{
				Layers: []ocispec.Descriptor{
					{Digest: digestAlpha},
					{Digest: digestBeta},
				},
			},
			cfg: oci.ConfigGit{
				Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{
					tagRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		err := m.UpdateRef(t.Context(), plumbing.NewHashReference(tagRefName, plumbing.NewHash(commitBeta)), digestBeta)

		assert.NoError(t, err)
		assert.Equal(t, m.cfg.Tags[tagRefName].Commit, commitBeta)
		assert.Equal(t, m.cfg.Tags[tagRefName].Layer, digestBeta)
	})

	t.Run("Unsupported Ref Type", func(t *testing.T) {
		m := &model{
			man: ocispec.Manifest{
				Layers: []ocispec.Descriptor{
					{Digest: digestAlpha},
					{Digest: digestBeta},
				},
			},
			cfg: oci.ConfigGit{
				Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{
					tagRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		err := m.UpdateRef(t.Context(), plumbing.NewHashReference(unsupportedRef, plumbing.NewHash(commitBeta)), digestBeta)

		assert.ErrorIs(t, err, ErrUnsupportedReferenceType)
	})

	t.Run("Layer DNE", func(t *testing.T) {
		m := &model{
			man: ocispec.Manifest{
				Layers: []ocispec.Descriptor{
					{Digest: digestAlpha},
				},
			},
			cfg: oci.ConfigGit{
				Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
					headRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		err := m.UpdateRef(t.Context(), plumbing.NewHashReference(headRefName, plumbing.NewHash(commitBeta)), digestDNE)

		assert.ErrorIs(t, err, errLayerNotInManifest)
	})
}

func Test_model_ResolveRef(t *testing.T) {
	const (
		digestAlpha = digest.Digest("sha256:ffbeaa9e113a29d9fc4f58f821e16f594e332033b277ea829eafab12ba148589")
		digestBeta  = digest.Digest("sha256:60290b69da490356c62dc190efe44ca597ec538f792c2908a8a7ec352dc13e5e")
		digestDNE   = digest.Digest("sha256:84290ebe369d11cd880bfa160ef8cbb7c0fc03b8093895534f30263b78f030b2")

		commitAlpha = "eaba08b8fae96b96fe68d88dd311ffb8ca22ba74"
		commitBeta  = "9f9daae4bb300543116a1508cd9ed87bafd9d5fc"
	)

	var (
		headRefName    = plumbing.NewBranchReferenceName("branchfoo")
		tagRefName     = plumbing.NewTagReferenceName("tagbar")
		unsupportedRef = plumbing.NewNoteReferenceName("notesfoobar")
	)

	t.Run("Success - Branch Ref", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
					headRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		gotFullRef, gotLayer, err := m.ResolveRef(t.Context(), headRefName)

		expectedFullRef := plumbing.NewHashReference(headRefName, plumbing.NewHash(commitAlpha))
		assert.NoError(t, err)
		assert.Equal(t, expectedFullRef, gotFullRef)
		assert.Equal(t, digestAlpha, gotLayer)
	})

	t.Run("Success - Tag Ref", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{
					tagRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		gotFullRef, gotLayer, err := m.ResolveRef(t.Context(), tagRefName)

		expectedFullRef := plumbing.NewHashReference(tagRefName, plumbing.NewHash(commitAlpha))
		assert.NoError(t, err)
		assert.Equal(t, expectedFullRef, gotFullRef)
		assert.Equal(t, digestAlpha, gotLayer)
	})

	t.Run("Unsupported Ref Type", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
					headRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		gotFullRef, gotLayer, err := m.ResolveRef(t.Context(), unsupportedRef)

		assert.ErrorIs(t, err, ErrUnsupportedReferenceType)
		assert.Nil(t, gotFullRef)
		assert.Equal(t, digest.Digest(""), gotLayer)
	})

	t.Run("Ref DNE", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
					headRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		nonexistantBranchRefName := plumbing.NewBranchReferenceName("nonexistent")

		gotFullRef, gotLayer, err := m.ResolveRef(t.Context(), nonexistantBranchRefName)

		assert.ErrorIs(t, err, ErrReferenceNotFound)
		assert.Nil(t, gotFullRef)
		assert.Equal(t, digest.Digest(""), gotLayer)
	})
}

func Test_model_DeleteRef(t *testing.T) {
	const (
		digestAlpha = digest.Digest("sha256:ffbeaa9e113a29d9fc4f58f821e16f594e332033b277ea829eafab12ba148589")
		digestBeta  = digest.Digest("sha256:60290b69da490356c62dc190efe44ca597ec538f792c2908a8a7ec352dc13e5e")
		digestDNE   = digest.Digest("sha256:84290ebe369d11cd880bfa160ef8cbb7c0fc03b8093895534f30263b78f030b2")

		commitAlpha = "eaba08b8fae96b96fe68d88dd311ffb8ca22ba74"
		commitBeta  = "9f9daae4bb300543116a1508cd9ed87bafd9d5fc"
	)

	var (
		headRefName    = plumbing.NewBranchReferenceName("branchfoo")
		tagRefName     = plumbing.NewTagReferenceName("tagbar")
		unsupportedRef = plumbing.NewNoteReferenceName("notesfoobar")
	)

	t.Run("Success - Branch Ref", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
					headRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		err := m.DeleteRef(t.Context(), headRefName)

		assert.NoError(t, err)

		expected := oci.ConfigGit{
			Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{},
		}

		assert.NotNil(t, m.cfg.Heads)
		assert.Equal(t, expected, m.cfg)
	})

	t.Run("Success - Tag Ref", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{
					tagRefName: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		err := m.DeleteRef(t.Context(), tagRefName)

		assert.NoError(t, err)

		expected := oci.ConfigGit{
			Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{},
		}

		assert.NotNil(t, m.cfg.Tags)
		assert.Equal(t, expected, m.cfg)
	})

	t.Run("Unsupported Ref Type", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
					unsupportedRef: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
				},
			},
		}

		err := m.DeleteRef(t.Context(), unsupportedRef)

		assert.ErrorIs(t, err, ErrUnsupportedReferenceType)

		// TODO: consider searching for the unsupported reference in case it got
		// added accidentally
		expected := oci.ConfigGit{
			Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
				unsupportedRef: {
					Commit: commitAlpha,
					Layer:  digestAlpha,
				},
			},
		}

		assert.NotNil(t, m.cfg.Heads)
		assert.Equal(t, expected, m.cfg)
	})

	t.Run("Ref DNE", func(t *testing.T) {
		m := &model{
			// Also testing how it behaves if maps are nil
			cfg: oci.ConfigGit{},
		}

		err := m.DeleteRef(t.Context(), tagRefName)
		assert.NoError(t, err)
		err = m.DeleteRef(t.Context(), headRefName)
		assert.NoError(t, err)
	})
}

// addCommit is a helper function for creating a git commit in a test repository.
// func addCommit(t *testing.T, dir string, repo *git.Repository, randData string) *object.Commit {
// 	t.Helper()

// 	w, err := repo.Worktree()
// 	assert.NoError(t, err)

// 	filename := filepath.Join(dir, "example-git-file")
// 	err = os.WriteFile(filename, []byte(randData), 0644)
// 	assert.NoError(t, err)

// 	_, err = w.Add("example-git-file")
// 	assert.NoError(t, err)

// 	commit, err := w.Commit("test commit", &git.CommitOptions{
// 		Author: &object.Signature{
// 			Name:  "John Doe",
// 			Email: "john@doe.org",
// 			When:  time.Now(),
// 		},
// 	})

// 	assert.NoError(t, err)

// 	commitHash, err := repo.CommitObject(commit)
// 	assert.NoError(t, err)

// 	return commitHash
// }

// TODO: finish me
// func Test_model_CommitExists(t *testing.T) {

// 	newCommit :=

// 		t.Run("Exists", func(t *testing.T) {
// 			dir := t.TempDir()
// 			repo, err := git.PlainInit(dir, false)
// 			assert.NoError(t, err)

// 			err = repo.CreateBranch(&config.Branch{Name: "foo"})
// 			assert.NoError(t, err)

// 		})
// }

func Test_model_sortRefsByLayer(t *testing.T) {
	const (
		digestAlpha = digest.Digest("sha256:ffbeaa9e113a29d9fc4f58f821e16f594e332033b277ea829eafab12ba148589")
		digestBeta  = digest.Digest("sha256:60290b69da490356c62dc190efe44ca597ec538f792c2908a8a7ec352dc13e5e")
		digestDNE   = digest.Digest("sha256:84290ebe369d11cd880bfa160ef8cbb7c0fc03b8093895534f30263b78f030b2")

		commitAlpha = "eaba08b8fae96b96fe68d88dd311ffb8ca22ba74"
		commitBeta  = "9f9daae4bb300543116a1508cd9ed87bafd9d5fc"
	)

	var (
		headRefName1 = plumbing.NewBranchReferenceName("branchfoo1")
		headRefName2 = plumbing.NewBranchReferenceName("branchfoo2")
		tagRefName1  = plumbing.NewTagReferenceName("tagbar1")
		tagRefName2  = plumbing.NewTagReferenceName("tagbar2")
	)

	t.Run("Sort Heads", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Heads: map[plumbing.ReferenceName]oci.ReferenceInfo{
					headRefName1: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
					headRefName2: {
						Commit: commitBeta,
						Layer:  digestBeta,
					},
				},
			},
		}

		m.sortRefsByLayer()

		expected := map[digest.Digest][]plumbing.Hash{
			digestAlpha: {plumbing.NewHash(commitAlpha)},
			digestBeta:  {plumbing.NewHash(commitBeta)},
		}

		assert.Equal(t, expected, m.refsByLayer)
	})

	t.Run("Sort - Tags", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Tags: map[plumbing.ReferenceName]oci.ReferenceInfo{
					tagRefName1: {
						Commit: commitAlpha,
						Layer:  digestAlpha,
					},
					tagRefName2: {
						Commit: commitBeta,
						Layer:  digestBeta,
					},
				},
			},
		}

		m.sortRefsByLayer()

		expected := map[digest.Digest][]plumbing.Hash{
			digestAlpha: {plumbing.NewHash(commitAlpha)},
			digestBeta:  {plumbing.NewHash(commitBeta)},
		}

		assert.Equal(t, expected, m.refsByLayer)
	})
}

func Test_model_HeadRefs(t *testing.T) {
	const (
		digestAlpha = digest.Digest("sha256:ffbeaa9e113a29d9fc4f58f821e16f594e332033b277ea829eafab12ba148589")
		commitAlpha = "eaba08b8fae96b96fe68d88dd311ffb8ca22ba74"
	)

	var (
		refAlpha = plumbing.NewBranchReferenceName("branchfoo")
		refBeta  = plumbing.NewBranchReferenceName("branchbar")
	)

	t.Run("Success", func(t *testing.T) {
		expected := map[plumbing.ReferenceName]oci.ReferenceInfo{
			refAlpha: {
				Commit: commitAlpha,
				Layer:  digestAlpha,
			},
			refBeta: {
				Commit: commitAlpha,
				Layer:  digestAlpha,
			},
		}

		m := &model{
			cfg: oci.ConfigGit{
				Heads: expected,
			},
		}

		got := m.HeadRefs()

		assert.Equal(t, expected, got)
	})

	t.Run("Nil", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Heads: nil,
			},
		}

		got := m.HeadRefs()

		assert.NotNil(t, got)
		assert.Equal(t, 0, len(got))
	})
}

func Test_model_TagRefs(t *testing.T) {
	const (
		digestAlpha = digest.Digest("sha256:ffbeaa9e113a29d9fc4f58f821e16f594e332033b277ea829eafab12ba148589")
		commitAlpha = "eaba08b8fae96b96fe68d88dd311ffb8ca22ba74"
	)

	var (
		refAlpha = plumbing.NewTagReferenceName("tagfoo")
		refBeta  = plumbing.NewTagReferenceName("tagbar")
	)

	t.Run("Success", func(t *testing.T) {
		expected := map[plumbing.ReferenceName]oci.ReferenceInfo{
			refAlpha: {
				Commit: commitAlpha,
				Layer:  digestAlpha,
			},
			refBeta: {
				Commit: commitAlpha,
				Layer:  digestAlpha,
			},
		}

		m := &model{
			cfg: oci.ConfigGit{
				Tags: expected,
			},
		}

		got := m.TagRefs()

		assert.Equal(t, expected, got)
	})

	t.Run("Nil", func(t *testing.T) {
		m := &model{
			cfg: oci.ConfigGit{
				Tags: nil,
			},
		}

		got := m.TagRefs()

		assert.NotNil(t, got)
		assert.Equal(t, 0, len(got))
	})
}
