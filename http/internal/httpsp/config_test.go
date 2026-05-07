package httpsp

import (
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/http/httpspi"
)

func TestNormalizeClientConfigFillsDefaults(t *testing.T) {
	cfg, err := normalizeClientConfig(httpspi.ClientConfig{
		BaseURL: "https://api.example.com",
		Retry: httpspi.RetryConfig{
			MaxRetries: 2,
		},
	})
	if err != nil {
		t.Fatalf("normalizeClientConfig: %v", err)
	}
	if cfg.DefaultTimeout != defaultClientTimeout {
		t.Fatalf("expected default timeout %s, got %s", defaultClientTimeout, cfg.DefaultTimeout)
	}
	if cfg.Retry.Delay != defaultRetryDelay {
		t.Fatalf("expected default retry delay %s, got %s", defaultRetryDelay, cfg.Retry.Delay)
	}
}

func TestNormalizeClientConfigCopiesHeaders(t *testing.T) {
	headers := map[string]string{"X-App": "order-service"}
	cfg, err := normalizeClientConfig(httpspi.ClientConfig{
		BaseURL:        "https://api.example.com",
		DefaultHeaders: headers,
	})
	if err != nil {
		t.Fatalf("normalizeClientConfig: %v", err)
	}

	headers["X-App"] = "mutated"
	if cfg.DefaultHeaders["X-App"] != "order-service" {
		t.Fatalf("expected copied headers, got %#v", cfg.DefaultHeaders)
	}
}

func TestNormalizeClientConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		cfg  httpspi.ClientConfig
	}{
		{
			name: "base url missing scheme",
			cfg:  httpspi.ClientConfig{BaseURL: "service.internal"},
		},
		{
			name: "negative timeout",
			cfg:  httpspi.ClientConfig{DefaultTimeout: -time.Second},
		},
		{
			name: "negative retry count",
			cfg:  httpspi.ClientConfig{Retry: httpspi.RetryConfig{MaxRetries: -1}},
		},
		{
			name: "negative retry delay",
			cfg:  httpspi.ClientConfig{Retry: httpspi.RetryConfig{Delay: -time.Millisecond}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := normalizeClientConfig(tt.cfg); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestNormalizeClientConfigsRejectsEmptyName(t *testing.T) {
	_, err := normalizeClientConfigs(httpspi.ClientConfigs{
		"": {BaseURL: "https://api.example.com"},
	})
	if err == nil {
		t.Fatalf("expected empty name error")
	}
}
