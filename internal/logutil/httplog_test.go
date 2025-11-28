package logutil

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/act3-ai/go-common/pkg/logger"
	"github.com/act3-ai/go-common/pkg/redact"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/act3-ai/gnoci/internal/mocks/httpmock"
)

const (
	redactedAuthHeader      = "foobarfoobar"
	redactedCookieHeader    = "barfoobarfoo"
	redactedSetCookieHeader = "foofoobarbar"
	redactedLocationHeader  = "https://example.com/path?token=123"
)

func Test_loggingTransport_RoundTrip(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rtMock := httpmock.NewMockRoundTripper(ctrl)

		var (
			reqBodyContents  = "request foo"
			respBodyContents = "response bar"
		)

		rtMock.EXPECT().
			RoundTrip(gomock.Any()).
			DoAndReturn(func(req *http.Request) (*http.Response, error) {
				// ensure request itself not redacted
				assert.Equal(t, redactedAuthHeader, req.Header.Get("Authorization"))
				assert.Equal(t, redactedCookieHeader, req.Header.Get("Cookie"))

				// ensure readable body
				buf := new(bytes.Buffer)
				_, err := io.Copy(buf, req.Body)
				assert.NoError(t, err)
				err = req.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, reqBodyContents, buf.String())

				respBody := io.NopCloser(strings.NewReader(respBodyContents))
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       respBody,
					Header: http.Header{
						"Location":   []string{redactedLocationHeader},
						"Set-Cookie": []string{redactedSetCookieHeader},
					},
				}

				return resp, nil
			})

		req := &http.Request{
			Method: http.MethodPost,
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/bar",
				User:     url.UserPassword("user", "pass"),
				RawQuery: "secret=true",
			},
			Header: http.Header{
				"Authorization": []string{redactedAuthHeader},
				"Cookie":        []string{redactedCookieHeader},
			},
			Body:          io.NopCloser(strings.NewReader(reqBodyContents)),
			ContentLength: int64(len(reqBodyContents)),
		}

		// init logger
		logOut := new(bytes.Buffer)
		options := &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug - 8,
		}
		handler := slog.NewJSONHandler(logOut, options)
		log := slog.New(handler)

		// inject logger into ctx
		ctx := logger.NewContext(t.Context(), log)
		req = req.WithContext(ctx)

		transport := &LoggingTransport{Base: rtMock}
		gotResp, err := transport.RoundTrip(req)
		defer func() {
			err := gotResp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, gotResp.StatusCode)

		// Validate the response header was restored after redaction
		assert.Equal(t, redactedSetCookieHeader, gotResp.Header.Get("Set-Cookie"))
		assert.Equal(t, redactedLocationHeader, gotResp.Header.Get("Location"))

		logs, err := io.ReadAll(logOut)
		assert.NoError(t, err)

		t.Logf("logs: %s", logs)
		assert.Equal(t, 3, bytes.Count(logs, []byte(redact.Redacted)))
		assert.False(t, bytes.Contains(logs, []byte("?token=123")))
	})
}

func Test_redactHTTPHeaders(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		headers := map[string][]string{
			"Authorization": {redactedAuthHeader},
			"Cookie":        {redactedCookieHeader},
			"Set-Cookie":    {redactedSetCookieHeader},
			"Location":      {redactedLocationHeader},
		}

		redactHTTPHeaders(headers)

		assert.Equal(t, redact.Redacted, headers["Authorization"][0])
		assert.Equal(t, redact.Redacted, headers["Cookie"][0])
		assert.Equal(t, redact.Redacted, headers["Set-Cookie"][0])
		assert.False(t, strings.Contains(headers["Location"][0], "?token=123"))
	})
}
