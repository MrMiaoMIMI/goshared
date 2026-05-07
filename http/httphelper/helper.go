// Package httphelper provides config-driven factory functions for creating
// httpspi.Client instances.
//
// Keep HTTP client settings in application configuration and create clients
// with NewClient or NewClients during service startup. httphelper intentionally
// does not expose WithXxx option functions; request-specific values are set on
// httpspi.Client.New() when sending a request.
package httphelper

import (
	"github.com/MrMiaoMIMI/goshared/http/httpspi"
	"github.com/MrMiaoMIMI/goshared/http/internal/httpsp"
)

// NewClient creates a new httpspi.Client from cfg.
//
// Configs are plain structs with json/yaml tags, so callers can keep base URL,
// timeout, default headers, and retry settings in application config files.
//
// Example:
//
//	client, err := httphelper.NewClient(cfg.HTTP.Clients["user_service"])
func NewClient(cfg httpspi.ClientConfig) (httpspi.Client, error) {
	return httpsp.NewHTTPClient(cfg)
}

// NewClients creates named clients from ClientConfigs.
//
// Use this when the application config manages multiple upstream services.
func NewClients(configs httpspi.ClientConfigs) (httpspi.Clients, error) {
	return httpsp.NewHTTPClients(configs)
}
