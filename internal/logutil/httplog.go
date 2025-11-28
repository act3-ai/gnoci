// Package logutil provides logging convenience functions.
package logutil

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"github.com/act3-ai/go-common/pkg/logger"
	"github.com/act3-ai/go-common/pkg/redact"
)

// file copied from https://github.com/act3-ai/data-tool/blob/v1.16.1/internal/httplogger/logging.go
// TODO: is there really no client-based implementation of something like this? There are some, but none are "good"

var requestNumber atomic.Int64

// LoggingTransport logs to the request's context.
// The output can be processed by jq to format it nicely.
type LoggingTransport struct {
	Base http.RoundTripper
}

// RoundTrip logs http requests and reponses while redacting sensistive information.
func (s *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	log := logger.V(logger.FromContext(ctx).WithGroup("http").With("requestID", requestNumber.Add(1)), 8)
	var err error

	// TODO: We can avoid double storing the body in memory if we add our own DumpRequestOut. For now,
	// 20KiB is acceptable.
	enabled := log.Enabled(ctx, slog.LevelInfo) // true if verbosity = 16
	if enabled {
		req := req.Clone(ctx)
		// redact the URL credentials and query string (S3 signed URLs have credentials there)
		req.URL.User = nil
		req.URL.RawQuery = ""
		redactHTTPHeaders(req.Header)

		reqBytes, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			log.ErrorContext(ctx, "Failed to dump the HTTP request", "error", err.Error())
		} else {
			log.InfoContext(ctx, "HTTP Request", "contents", string(reqBytes))
		}
	}

	resp, err := s.Base.RoundTrip(req)
	// err is returned after dumping the response

	// need to check if response is nil so that go doesn't panic w/ segfault
	if resp != nil && enabled {
		savedHeaders := resp.Header.Clone()
		redactHTTPHeaders(resp.Header)
		// TODO redact the body of the auth response
		// for now we always omit the body to be conservative
		respBytes, err := httputil.DumpResponse(resp, false) // resp.ContentLength < maxSize)
		if err != nil {
			log.ErrorContext(ctx, "Failed to dump the HTTP response", "error", err.Error())
		} else {
			log.InfoContext(ctx, "HTTP Response", "contents", string(respBytes))
		}

		// restore then
		resp.Header = savedHeaders
	}

	return resp, err //nolint:wrapcheck
}

var redactedHeaders = []string{
	"Authorization",
	"Cookie", // probably not needed but why not
	"Set-Cookie",
}

// redact http headers in place.
func redactHTTPHeaders(hdrs http.Header) {
	// redact headers Authorization, Cookie, Set-Cookie
	// redact query params of Location headers
	for _, h := range redactedHeaders {
		values := hdrs.Values(h)
		for i, value := range values {
			values[i] = redact.String(value)
		}
	}

	values := hdrs.Values("Location")
	for i, value := range values {
		values[i] = redactURL(value)
	}
}

// redact the URL inplace removing user credentials and query string params.
func redactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	u.User = nil
	u.RawQuery = ""
	return u.String()
}
