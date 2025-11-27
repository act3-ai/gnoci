// Package refcomp provides utilities for comparing local and remote git references.
package refcomp

import (
	"reflect"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/act3-ai/gnoci/internal/git"
	"github.com/act3-ai/gnoci/internal/mocks/gitmock"
	"github.com/act3-ai/gnoci/internal/mocks/modelmock"
	"github.com/act3-ai/gnoci/internal/ociutil/model"
)

func TestNewCachedRefComparer(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		repoMock := gitmock.NewMockRepository(ctrl)
		modelMock := modelmock.NewMockModeler(ctrl)

		rc := NewCachedRefComparer(repoMock, modelMock)
		assert.NotNil(t, rc)

		rcConcrete, ok := rc.(*refCompare)
		assert.True(t, ok)
		assert.NotNil(t, rcConcrete.local)
		assert.NotNil(t, rcConcrete.remote)
		assert.NotNil(t, rcConcrete.refs)
	})
}

func Test_refCompare_Compare(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			localRef  = plumbing.NewBranchReferenceName("local")
			remoteRef = plumbing.NewBranchReferenceName("remote")
		)

		ctrl := gomock.NewController(t)

		repoMock := gitmock.NewMockRepository(ctrl)
		modelMock := modelmock.NewMockModeler(ctrl)

		rc := &refCompare{
			local:  repoMock,
			remote: modelMock,
			refs:   make(map[plumbing.ReferenceName]RefPair, 0),
		}

		rc.Compare(t.Context(), false)
	})
}

func Test_refCompare_compare(t *testing.T) {
	type fields struct {
		local  git.Repository
		remote model.Modeler
		refs   map[plumbing.ReferenceName]RefPair
	}
	type args struct {
		force     bool
		localRef  *plumbing.Reference
		remoteRef *plumbing.Reference
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    RefPair
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &refCompare{
				local:  tt.fields.local,
				remote: tt.fields.remote,
				refs:   tt.fields.refs,
			}
			got, err := rc.compare(tt.args.force, tt.args.localRef, tt.args.remoteRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("refCompare.compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("refCompare.compare() = %v, want %v", got, tt.want)
			}
		})
	}
}
