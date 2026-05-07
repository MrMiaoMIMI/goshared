package httpsp

import (
	"fmt"
	"net/url"
	"time"

	"github.com/MrMiaoMIMI/goshared/http/httpspi"
)

const (
	defaultClientTimeout = 30 * time.Second
	defaultRetryDelay    = 100 * time.Millisecond
)

func normalizeClientConfig(cfg httpspi.ClientConfig) (httpspi.ClientConfig, error) {
	if cfg.BaseURL != "" {
		parsed, err := url.Parse(cfg.BaseURL)
		if err != nil {
			return httpspi.ClientConfig{}, fmt.Errorf("http: invalid base_url %q: %w", cfg.BaseURL, err)
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return httpspi.ClientConfig{}, fmt.Errorf("http: base_url must include scheme and host")
		}
	}

	if cfg.DefaultTimeout < 0 {
		return httpspi.ClientConfig{}, fmt.Errorf("http: default_timeout must be >= 0")
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = defaultClientTimeout
	}

	if cfg.Retry.MaxRetries < 0 {
		return httpspi.ClientConfig{}, fmt.Errorf("http: retry.max_retries must be >= 0")
	}
	if cfg.Retry.Delay < 0 {
		return httpspi.ClientConfig{}, fmt.Errorf("http: retry.delay must be >= 0")
	}
	if cfg.Retry.MaxRetries > 0 && cfg.Retry.Delay < defaultRetryDelay {
		cfg.Retry.Delay = defaultRetryDelay
	}

	if cfg.DefaultHeaders != nil {
		headers := make(map[string]string, len(cfg.DefaultHeaders))
		for k, v := range cfg.DefaultHeaders {
			headers[k] = v
		}
		cfg.DefaultHeaders = headers
	}

	return cfg, nil
}

func normalizeClientConfigs(configs httpspi.ClientConfigs) (httpspi.ClientConfigs, error) {
	normalized := make(httpspi.ClientConfigs, len(configs))
	for name, cfg := range configs {
		if name == "" {
			return nil, fmt.Errorf("http: client name is required")
		}
		cfg, err := normalizeClientConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("http: client %q: %w", name, err)
		}
		normalized[name] = cfg
	}
	return normalized, nil
}
