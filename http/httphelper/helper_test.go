package httphelper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/http/httpspi"
)

func TestNewClientCreatesClientFromConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := json.Marshal(map[string]string{
			"header": r.Header.Get("X-App"),
		})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	headers := map[string]string{"X-App": "order-service"}
	client, err := NewClient(httpspi.ClientConfig{
		BaseURL:        server.URL,
		DefaultTimeout: time.Second,
		DefaultHeaders: headers,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	headers["X-App"] = "mutated"

	var resp map[string]string
	_, err = client.New().Get("/").Request(context.Background(), &resp, nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if resp["header"] != "order-service" {
		t.Fatalf("expected copied default header, got %q", resp["header"])
	}
}

func TestNewClientsCreatesNamedClients(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	clients, err := NewClients(httpspi.ClientConfigs{
		"user_service": {
			BaseURL: server.URL,
		},
		"order_service": {
			BaseURL: server.URL,
		},
	})
	if err != nil {
		t.Fatalf("NewClients: %v", err)
	}
	if clients["user_service"] == nil || clients["order_service"] == nil {
		t.Fatalf("expected named clients to be created: %#v", clients)
	}
}

func TestNewClientsRejectsInvalidNamedConfig(t *testing.T) {
	_, err := NewClients(httpspi.ClientConfigs{
		"user_service": {
			BaseURL: "service.internal",
		},
	})
	if err == nil {
		t.Fatalf("expected invalid named client config error")
	}
}

func TestNewClientRejectsInvalidConfig(t *testing.T) {
	_, err := NewClient(httpspi.ClientConfig{
		BaseURL: "service.internal",
	})
	if err == nil {
		t.Fatalf("expected invalid base_url error")
	}
}
