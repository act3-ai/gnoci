// Package httpmock mocks stdlib net/http.RoundTripper.
package httpmock

//go:generate go tool mockgen -typed -package httpmock -destination ./roundtrippermock.gen.go net/http RoundTripper
