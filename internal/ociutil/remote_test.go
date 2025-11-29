// Package ociutil provides utility functions for interacting with OCI registries.
package ociutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

var testRemote = registry.Reference{
	Registry:   "reg.example.com",
	Repository: "repo",
	Reference:  "tag",
}

func TestRepositoryOptions_defaulter(t *testing.T) {
	t.Run("No Defaulting", func(t *testing.T) {
		expectedOpts := RepositoryOptions{
			UserAgent:     "foo",
			PlainHTTP:     true,
			NonCompliant:  true,
			RegistryCreds: credentials.NewMemoryStore(),
		}

		opts := RepositoryOptions{
			UserAgent:     expectedOpts.UserAgent,
			PlainHTTP:     expectedOpts.PlainHTTP,
			NonCompliant:  expectedOpts.NonCompliant,
			RegistryCreds: expectedOpts.RegistryCreds,
		}

		opts.defaulter(t.Context())
		assert.Equal(t, expectedOpts, opts)
	})

	t.Run("Default UserAgent", func(t *testing.T) {
		expectedOpts := RepositoryOptions{
			UserAgent:     gnociUserAgent,
			PlainHTTP:     true,
			NonCompliant:  true,
			RegistryCreds: credentials.NewMemoryStore(),
		}

		opts := RepositoryOptions{
			UserAgent:     "",
			PlainHTTP:     expectedOpts.PlainHTTP,
			NonCompliant:  expectedOpts.NonCompliant,
			RegistryCreds: expectedOpts.RegistryCreds,
		}

		opts.defaulter(t.Context())
		assert.Equal(t, expectedOpts, opts)
	})

	t.Run("Default Credential Store", func(t *testing.T) {
		storeOpts := credentials.StoreOptions{}
		expectedCredStore, err := credentials.NewStoreFromDocker(storeOpts)
		assert.NoError(t, err)

		expectedOpts := RepositoryOptions{
			UserAgent:     "foo",
			PlainHTTP:     true,
			NonCompliant:  true,
			RegistryCreds: expectedCredStore,
		}

		opts := RepositoryOptions{
			UserAgent:     expectedOpts.UserAgent,
			PlainHTTP:     expectedOpts.PlainHTTP,
			NonCompliant:  expectedOpts.NonCompliant,
			RegistryCreds: nil,
		}

		opts.defaulter(t.Context())
		assert.Equal(t, expectedOpts, opts)
	})
}

func TestNewGraphTarget(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		gt, err := NewGraphTarget(t.Context(), testRemote, &RepositoryOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, gt)
	})
}

func Test_create(t *testing.T) {
	t.Run("Default Auth Cache", func(t *testing.T) {
		opts := RepositoryOptions{
			NonCompliant: false,
		}

		gt, err := create(t.Context(), testRemote, &opts)
		assert.NoError(t, err)
		assert.NotNil(t, gt)

		repo, ok := gt.(*remote.Repository)
		assert.True(t, ok)
		assert.NotNil(t, repo)

		client, ok := repo.Client.(*auth.Client)
		assert.True(t, ok)
		assert.NotNil(t, client)

		assert.Equal(t, auth.DefaultCache, client.Cache)
	})

	t.Run("NonCompliant Auth Cache", func(t *testing.T) {
		opts := RepositoryOptions{
			NonCompliant: true,
		}

		gt, err := create(t.Context(), testRemote, &opts)
		assert.NoError(t, err)
		assert.NotNil(t, gt)

		repo, ok := gt.(*remote.Repository)
		assert.True(t, ok)
		assert.NotNil(t, repo)

		client, ok := repo.Client.(*auth.Client)
		assert.True(t, ok)
		assert.NotNil(t, client)

		assert.Equal(t, auth.NewSingleContextCache(), client.Cache)
	})

	t.Run("PlainHTTP Disabled", func(t *testing.T) {
		opts := RepositoryOptions{
			PlainHTTP: false,
		}

		gt, err := create(t.Context(), testRemote, &opts)
		assert.NoError(t, err)
		assert.NotNil(t, gt)

		repo, ok := gt.(*remote.Repository)
		assert.True(t, ok)
		assert.NotNil(t, repo)

		assert.False(t, repo.PlainHTTP)
	})

	t.Run("PlainHTTP Enabled", func(t *testing.T) {
		opts := RepositoryOptions{
			PlainHTTP: true,
		}

		gt, err := create(t.Context(), testRemote, &opts)
		assert.NoError(t, err)
		assert.NotNil(t, gt)

		repo, ok := gt.(*remote.Repository)
		assert.True(t, ok)
		assert.NotNil(t, repo)

		assert.True(t, repo.PlainHTTP)
	})
}

func Test_newHTTPClientWithOps(t *testing.T) {
	type args struct {
		hostName       string
		customCertPath string
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Client
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newHTTPClientWithOps(tt.args.hostName, tt.args.customCertPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("newHTTPClientWithOps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newHTTPClientWithOps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_resolveTLSCertLocation(t *testing.T) {
	t.Run("Success On First", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "foo.pem")
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		assert.NoError(t, err)
		defer f.Close()

		_, err = f.WriteString("foobarfoobar")
		assert.NoError(t, err)
		err = f.Close()
		assert.NoError(t, err)

		certPath, err := resolveTLSCertLocation([]string{filePath})
		assert.NoError(t, err)
		assert.Equal(t, filePath, certPath)
	})

	t.Run("Success On Not First", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "foo.pem")
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		assert.NoError(t, err)
		defer f.Close()

		_, err = f.WriteString("foobarfoobar")
		assert.NoError(t, err)
		err = f.Close()
		assert.NoError(t, err)

		dne1 := filepath.Join(tmpDir, "dne1")
		dne2 := filepath.Join(tmpDir, "dne2")

		certPath, err := resolveTLSCertLocation([]string{dne1, dne2, filePath})
		assert.NoError(t, err)
		assert.Equal(t, filePath, certPath)
	})

	t.Run("Path Not Exist", func(t *testing.T) {
		filepath := filepath.Join(t.TempDir(), "path", "dne")
		certPath, err := resolveTLSCertLocation([]string{filepath})
		assert.NoError(t, err)
		assert.Equal(t, "", certPath)
	})
}

func Test_fetchCertsFromLocation(t *testing.T) {
	t.Run("Custom Certificate Directory", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		assert.NoError(t, err)

		// Certificate template
		tmpl := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName:   "localhost",
				Organization: []string{"Test Org"},
			},
			NotBefore: time.Now().Add(-time.Hour),
			NotAfter:  time.Now().Add(24 * time.Hour),

			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		// Self-sign
		derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &privateKey.PublicKey, privateKey)
		assert.NoError(t, err)

		// PEM-encode cert
		certPEM := pem.EncodeToMemory(
			&pem.Block{Type: "CERTIFICATE", Bytes: derBytes},
		)

		// PEM-encode key
		keyPEM := pem.EncodeToMemory(
			&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)},
		)

		tmpDir := t.TempDir()

		certPath := filepath.Join(tmpDir, "cert.pem")
		keyPath := filepath.Join(tmpDir, "key.pem")

		err = os.WriteFile(certPath, certPEM, 0o600)
		assert.NoError(t, err)

		err = os.WriteFile(keyPath, keyPEM, 0o600)
		assert.NoError(t, err)

		// now for our actual test
		tlsCfg, err := fetchCertsFromLocation(tmpDir)
		assert.NoError(t, err)
		assert.NotNil(t, tlsCfg)
	})
}
