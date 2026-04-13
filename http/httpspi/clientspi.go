// Package httpspi defines the HTTP client SPI (Service Provider Interface).
// It contains stable public interfaces for building and sending HTTP requests.
package httpspi

import (
	"context"
	"net/http"
	"time"
)

// Client is used to generate and send net/http request.
// A Client is designed as a fluent builder — each setter returns the Client itself for chaining.
// Use New() to create a per-request copy from a shared base Client.
type Client interface {

	// New creates a shallow copy of this Client, inheriting base URL, default headers,
	// default timeout, ResponseDecoder, and the underlying http.Client.
	// Per-request state (method, path, body, query params, cookies) is reset.
	// Always call New() before building a new request to avoid state leakage.
	New() Client

	// ==================== Header ====================

	// Add adds the key, value pair in Headers, appending values for existing keys
	// to the key's values. Header keys are canonicalized.
	Add(key, value string) Client

	// Set sets the key, value pair in Headers, replacing existing values
	// associated with key. Header keys are canonicalized.
	Set(key, value string) Client

	// Header sets the Client's header, replacing the old values of the keys.
	Header(header http.Header) Client

	// HeaderMap sets the Client's headers, replacing the old values of the keys.
	HeaderMap(headers map[string]string) Client

	// AddCookies adds cookies to the request's header.
	// All cookies added will be written into the same line with Cookie header and separated by semicolon.
	// All cookies only contain the sanitized name and value using net/http sanitizeCookieName and sanitizeCookieValue methods
	// Example of the cookie header: Cookie: name1=value1; name2=value2
	AddCookies(cookie ...*http.Cookie) Client

	// ==================== URL ====================

	// Base sets the rawURL.
	Base(rawURL string) Client

	// QueryStruct appends the queryStruct to the request's URL queries.
	// The queryStruct argument should be a pointer to a struct with "url" tagged fields.
	// Fields tagged with `url:"name"` are encoded as query parameters.
	// Use `url:"name,omitempty"` to skip zero-value fields.
	QueryStruct(queryStruct any) Client

	// QueryParam appends a single query parameter to the request's URL.
	// Multiple calls with the same key will append multiple values.
	QueryParam(key, value string) Client

	// ==================== Method ====================

	// Get sets the request method to GET and sets the given pathURL.
	Get(pathURL string) Client

	// Post sets the request method to POST and sets the given pathURL.
	Post(pathURL string) Client

	// Head sets the request method to HEAD and sets the given pathURL.
	Head(pathURL string) Client

	// Put sets the request method to PUT and sets the given pathURL.
	Put(pathURL string) Client

	// Patch sets the request method to PATCH and sets the given pathURL.
	Patch(pathURL string) Client

	// Delete sets the request method to DELETE and sets the given pathURL.
	Delete(pathURL string) Client

	// Options sets the request method to OPTIONS and sets the given pathURL.
	Options(pathURL string) Client

	// ==================== Body ====================

	// BodyJSON sets the Client's Request body to the JSON encoded value of bodyJSON
	// and sets the Content-Type header to "application/json".
	BodyJSON(bodyJSON any) Client

	// BodyBytes sets the Client's Request body to the given raw bytes.
	BodyBytes(bytes []byte) Client

	// ==================== Send ====================

	// ResponseDecoder sets the Client's responseDecoder.
	// If not set, a default JSON response decoder is used.
	ResponseDecoder(decoder ResponseDecoder) Client

	// Receive creates a new HTTP request and returns the response.
	// Success responses (2XX) are decoded into the value pointed to by successV.
	// Otherwise, the response is decoded into failureV.
	// If the respective receiver is nil, then the response body will remain untouched,
	// and it's the caller's responsibility to completely read and close the response body.
	// A positive timeout applies a per-request deadline on top of ctx; zero means no extra deadline.
	Receive(ctx context.Context, timeout time.Duration, successV, failureV any) (*http.Response, error)

	// Request is a simplified version of Receive that uses the client's default timeout.
	// It drains and closes the response body automatically, returning a lightweight Response
	// that carries status code and headers without the body.
	// Success responses (2XX) are decoded into successV; non-2XX into failureV.
	// For non-2XX responses, a StatusError is also returned as the error
	// (failureV is still populated if non-nil).
	Request(ctx context.Context, successV, failureV any) (*Response, error)
}

// Response is a lightweight representation of an HTTP response returned by Client.Request.
// It carries the essential response metadata without the body, which is
// automatically drained and closed by Request.
type Response struct {
	StatusCode int
	Status     string
	Header     http.Header
}

// ResponseDecoder decodes http responses into struct values.
type ResponseDecoder interface {
	// Decode decodes the response body into the value pointed to by v.
	// Implementations should read the body but MUST NOT close it.
	Decode(resp *http.Response, v any) error
}
