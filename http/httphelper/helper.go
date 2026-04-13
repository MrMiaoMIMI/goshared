// Package httphelper provides factory functions for creating httpspi.Client instances.
// Callers use this package to construct clients without importing internal packages.
package httphelper

import (
	"net/http"
	"time"

	"github.com/MrMiaoMIMI/goshared/http/httpspi"
	"github.com/MrMiaoMIMI/goshared/http/internal/httpsp"
)

// ClientOption is a type alias so callers don't need to import the internal package.
type ClientOption = httpsp.ClientOption

// NewClient creates a new httpspi.Client with the given options.
//
// Example:
//
//	client := httphelper.NewClient(
//	    httphelper.WithBaseURL("https://api.example.com"),
//	    httphelper.WithDefaultTimeout(10 * time.Second),
//	)
func NewClient(opts ...ClientOption) httpspi.Client {
	return httpsp.NewHTTPClient(opts...)
}

// WithBaseURL sets the base URL shared by all requests from this client.
func WithBaseURL(baseURL string) ClientOption {
	return httpsp.WithBaseURL(baseURL)
}

// WithDefaultHeaders sets headers applied to every request built from this client.
func WithDefaultHeaders(headers map[string]string) ClientOption {
	return httpsp.WithDefaultHeaders(headers)
}

// WithDefaultTimeout sets the default per-request timeout used by Client.Request().
func WithDefaultTimeout(d time.Duration) ClientOption {
	return httpsp.WithDefaultTimeout(d)
}

// WithHTTPClient sets the underlying *http.Client used for transport.
func WithHTTPClient(c *http.Client) ClientOption {
	return httpsp.WithHTTPClient(c)
}
