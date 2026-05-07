package httpspi

import "time"

// ClientConfig is the user-facing configuration for httphelper.NewClient.
//
// Keep this struct in application config files and pass it to httphelper at
// startup. Request-specific values such as path, query, body, and auth headers
// should be set on Client.New() when sending a request.
//
// Example YAML:
//
//	http:
//	  clients:
//	    user_service:
//	      base_url: https://user.example.com
//	      default_timeout: 5s
//	      default_headers:
//	        X-App: order-service
//	      retry:
//	        max_retries: 2
//	        delay: 200ms
type ClientConfig struct {
	// BaseURL is joined with the request path passed to Get/Post/etc.
	// Leave it empty when every request uses an absolute URL.
	BaseURL string `json:"base_url" yaml:"base_url"`

	// DefaultTimeout is used by Client.Request. Zero uses 30 seconds.
	DefaultTimeout time.Duration `json:"default_timeout" yaml:"default_timeout"`

	// DefaultHeaders are copied into every new request.
	DefaultHeaders map[string]string `json:"default_headers" yaml:"default_headers"`

	// Retry controls retry behavior for network errors and 5xx responses.
	Retry RetryConfig `json:"retry" yaml:"retry"`
}

// ClientConfigs is a named collection of HTTP client configurations.
//
// Use it for config sections that manage multiple upstream HTTP services:
//
//	http:
//	  clients:
//	    user_service:
//	      base_url: https://user.example.com
//	    payment_service:
//	      base_url: https://payment.example.com
type ClientConfigs map[string]ClientConfig

// RetryConfig controls retries for transient HTTP failures.
type RetryConfig struct {
	// MaxRetries is the number of retries after the first attempt.
	// Zero disables retry.
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// Delay is the wait between retry attempts.
	// Zero uses 100 milliseconds when MaxRetries is greater than zero.
	Delay time.Duration `json:"delay" yaml:"delay"`
}
