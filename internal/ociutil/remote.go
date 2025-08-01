// Package ociutil provides utility functions for interacting with OCI registries.
package ociutil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/act3-ai/go-common/pkg/logger"
	"github.com/adrg/xdg"
	"golang.org/x/net/proxy"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// This file is a modification of the contents of https://github.com/act3-ai/data-tool/blob/v1.16.1/internal/registry/registry.go

// NewGraphTarget creates an oras.GraphTarget.
//
// TODO: Due to a need to support special use cases, we'll likely need to define a configuration file.
func NewGraphTarget(ctx context.Context, ref string) (oras.GraphTarget, error) {
	parsedRef, err := registry.ParseReference(ref)
	if err != nil {
		return nil, fmt.Errorf("invalid reference %s: %w", ref, err)
	}

	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential store: %w", err)
	}

	return create(ctx, parsedRef, false, "gitoci", credStore)
}

func create(ctx context.Context, ref registry.Reference, noncompliant bool, userAgent string, credStore credentials.Store) (oras.GraphTarget, error) {
	log := logger.FromContext(ctx)

	var cache auth.Cache
	if noncompliant {
		log.InfoContext(ctx, "noncompliant registry detected, using single context auth cache")
		cache = auth.NewSingleContextCache()
	} else {
		cache = auth.DefaultCache
	}

	c, err := newHTTPClientWithOps(ref.Registry, "")
	if err != nil {
		return nil, err
	}

	// create the endpoint registry object
	reg := &remote.Registry{
		RepositoryOptions: remote.RepositoryOptions{
			Client: &auth.Client{
				Client: c,
				Header: http.Header{
					"User-Agent": {userAgent},
				},
				Cache:      cache,
				Credential: credentials.Credential(credStore),
			},
			Reference:       ref,
			PlainHTTP:       (strings.HasPrefix(ref.Registry, "localhost") || strings.HasPrefix(ref.Registry, "127.0.0.1")), // HACK: we need a config
			SkipReferrersGC: true,
		},
	}

	repo, err := reg.Repository(ctx, ref.Repository)
	if err != nil {
		return nil, fmt.Errorf("creating registry repository: %w", err)
	}

	r, ok := repo.(*remote.Repository)
	if !ok {
		return nil, fmt.Errorf("error creating registry repository: %s", ref)
	}

	return oras.GraphTarget(r), nil
}

// if a nil TLS is passed, return a client with a logging transport wrapped in a retry transport.
// if a TLS config exists, search for TLS certs and append to client.
func newHTTPClientWithOps(hostName, customCertPath string) (*http.Client, error) {
	nd := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	// defaultTransport is a new instance of the default transport
	var defaultTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           nd.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	var certLocation string
	if customCertPath == "" {
		cLocation, err := resolveTLSCertLocation(getStandardCertLocations(hostName))
		if err != nil {
			return nil, err
		}
		certLocation = cLocation
	}

	ssl, err := fetchCertsFromLocation(certLocation)
	if err != nil {
		return nil, err
	}

	defaultTransport.TLSClientConfig = ssl

	// get the proxy from the environment
	dialer := proxy.FromEnvironment()

	if dialer != nil {
		defaultTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	}

	// log requests to the logger (if verbosity is high enough)
	lt := &loggingTransport{
		Base: defaultTransport,
	}

	// we still want retry
	rt := retry.NewTransport(lt)

	return &http.Client{
		Transport: rt,
	}, nil
}

// resolveTLSCertLocation first searches for the registry certs in containerd's default TLS config path.
// If it is not located there it falls back to docker's default TLS config path.
// If there is no cert repository it will return an empty string.
// More info on containerd: https://github.com/containerd/containerd/blob/main/docs/hosts.md
// More info on docker: https://docs.docker.com/engine/reference/commandline/dockerd/#insecure-registries
func resolveTLSCertLocation(paths []string) (string, error) {
	// locations to search for certs
	// containerdPath := filepath.Join("/etc/containerd/certs.d", hostName)
	// dockerPath := filepath.Join(xdg.Home, ".docker/certs.d", hostName)
	// etcDockerPath := filepath.Join("/etc/docker/certs.d", hostName)

	for _, certPath := range paths {
		_, err := os.Stat(certPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("error accessing the TLS certificates in %s: %w", certPath, err)
		}
		return certPath, nil
	}
	return "", nil
}

func fetchCertsFromLocation(certDir string) (*tls.Config, error) {
	certFilePath := filepath.Join(certDir, "cert.pem")
	keyFilePath := filepath.Join(certDir, "key.pem")
	caFilePath := filepath.Join(certDir, "ca.pem")

	tlscfg := &tls.Config{}

	// add system level certs
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("fetching system certs: %w", err)
	}

	if certDir != "" {
		// Load client cert
		cert, err := tls.LoadX509KeyPair(certFilePath, keyFilePath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("error reading the certificate and key files: %w", err)
			}
		}
		tlscfg.Certificates = []tls.Certificate{cert}

		// Load CA cert
		caCert, err := os.ReadFile(caFilePath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return tlscfg, nil
			}
			return nil, fmt.Errorf("error reading the caFile: %w", err)
		}

		// Only trust this CA for this host
		caCertPool.AppendCertsFromPEM(caCert)

	}

	tlscfg.RootCAs = caCertPool

	return tlscfg, nil
}

// Currently, there are three standard locations checked for TLS certificates in ace-dt (modeled after containerd's implementation).
// First we check the standard containerd location for certs in /etc/containerd/certs.d/{HOSTNAME}.
// If it is not located there, we follow containerd's fallback location checks in docker's 2 certificate locations, located in /etc/docker/certs.d/{HOSTNAME} and ~/.docker/certs.d/{HOSTNAME} respectively.
func getStandardCertLocations(hostName string) []string {
	return []string{
		filepath.Join("/etc/containerd/certs.d", hostName), filepath.Join("/etc/docker/certs.d", hostName), filepath.Join(xdg.Home, ".docker/certs.d", hostName),
	}
}
