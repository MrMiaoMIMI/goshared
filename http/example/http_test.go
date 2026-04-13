package example

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/http/httphelper"
	"github.com/MrMiaoMIMI/goshared/http/httpspi"
)

// ==================== Test Models ====================

type EchoResponse struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type SearchParams struct {
	Query string `url:"q"`
	Page  int    `url:"page"`
	Limit int    `url:"limit,omitempty"`
}

// ==================== Test Server ====================

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		query := make(map[string]string)
		for k, v := range r.URL.Query() {
			query[k] = v[0]
		}

		headers := make(map[string]string)
		for k, v := range r.Header {
			headers[k] = v[0]
		}

		resp := EchoResponse{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   query,
			Headers: headers,
			Body:    string(body),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Server-Version", "1.0")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{Code: 0, Message: "ok"})
	})

	mux.HandleFunc("/error/400", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Error-Code", "VALIDATION")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{Code: 400, Message: "bad request"})
	})

	mux.HandleFunc("/error/500", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{Code: 500, Message: "internal error"})
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{Code: 0, Message: "slow response"})
	})

	return httptest.NewServer(mux)
}

func newTestClient(baseURL string) httpspi.Client {
	return httphelper.NewClient(
		httphelper.WithBaseURL(baseURL),
		httphelper.WithDefaultTimeout(5*time.Second),
	)
}

// ==================== Tests ====================

func Test_GET_Simple(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	resp, err := client.New().Get("/echo").Request(context.Background(), &echo, nil)
	assertNoError(t, "GET /echo", err)
	assertEqual(t, "method", "GET", echo.Method)
	assertEqual(t, "path", "/echo", echo.Path)
	assertEqual(t, "resp.StatusCode", 200, resp.StatusCode)
}

func Test_POST_WithJSON(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	reqBody := CreateUserRequest{Name: "Alice", Email: "alice@example.com"}
	var echo EchoResponse
	resp, err := client.New().Post("/echo").BodyJSON(&reqBody).Request(context.Background(), &echo, nil)
	assertNoError(t, "POST /echo", err)
	assertEqual(t, "method", "POST", echo.Method)
	assertEqual(t, "resp.StatusCode", 200, resp.StatusCode)

	var gotBody CreateUserRequest
	json.Unmarshal([]byte(echo.Body), &gotBody)
	assertEqual(t, "body.name", "Alice", gotBody.Name)
	assertEqual(t, "body.email", "alice@example.com", gotBody.Email)
	assertEqual(t, "content-type", "application/json", echo.Headers["Content-Type"])
}

func Test_PUT_WithBodyBytes(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	_, err := client.New().Put("/echo").BodyBytes([]byte("raw data")).Request(context.Background(), &echo, nil)
	assertNoError(t, "PUT /echo", err)
	assertEqual(t, "method", "PUT", echo.Method)
	assertEqual(t, "body", "raw data", echo.Body)
}

func Test_DELETE(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	_, err := client.New().Delete("/echo").Request(context.Background(), &echo, nil)
	assertNoError(t, "DELETE /echo", err)
	assertEqual(t, "method", "DELETE", echo.Method)
}

func Test_PATCH(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	_, err := client.New().Patch("/echo").Request(context.Background(), &echo, nil)
	assertNoError(t, "PATCH /echo", err)
	assertEqual(t, "method", "PATCH", echo.Method)
}

func Test_HEAD(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	resp, err := client.New().Head("/echo").Receive(context.Background(), 5*time.Second, nil, nil)
	assertNoError(t, "HEAD /echo", err)
	defer resp.Body.Close()
	assertEqual(t, "status", 200, resp.StatusCode)
}

func Test_QueryParam(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	_, err := client.New().
		Get("/echo").
		QueryParam("q", "hello").
		QueryParam("page", "2").
		Request(context.Background(), &echo, nil)
	assertNoError(t, "GET /echo?q=hello&page=2", err)
	assertEqual(t, "query.q", "hello", echo.Query["q"])
	assertEqual(t, "query.page", "2", echo.Query["page"])
}

func Test_QueryStruct(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	params := SearchParams{Query: "goshared", Page: 1, Limit: 0}
	var echo EchoResponse
	_, err := client.New().
		Get("/echo").
		QueryStruct(&params).
		Request(context.Background(), &echo, nil)
	assertNoError(t, "GET /echo with QueryStruct", err)
	assertEqual(t, "query.q", "goshared", echo.Query["q"])
	assertEqual(t, "query.page", "1", echo.Query["page"])

	if _, exists := echo.Query["limit"]; exists {
		t.Fatal("expected limit to be omitted (omitempty + zero value)")
	}
}

func Test_CustomHeaders(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	_, err := client.New().
		Get("/echo").
		Set("X-Custom", "value1").
		HeaderMap(map[string]string{"X-Another": "value2"}).
		Request(context.Background(), &echo, nil)
	assertNoError(t, "GET with headers", err)
	assertEqual(t, "X-Custom", "value1", echo.Headers["X-Custom"])
	assertEqual(t, "X-Another", "value2", echo.Headers["X-Another"])
}

func Test_DefaultHeaders(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := httphelper.NewClient(
		httphelper.WithBaseURL(server.URL),
		httphelper.WithDefaultTimeout(5*time.Second),
		httphelper.WithDefaultHeaders(map[string]string{
			"Authorization": "Bearer test-token",
		}),
	)

	var echo EchoResponse
	_, err := client.New().Get("/echo").Request(context.Background(), &echo, nil)
	assertNoError(t, "GET with default headers", err)
	assertEqual(t, "Authorization", "Bearer test-token", echo.Headers["Authorization"])
}

func Test_Cookies(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	_, err := client.New().
		Get("/echo").
		AddCookies(
			&http.Cookie{Name: "session", Value: "abc123"},
			&http.Cookie{Name: "lang", Value: "en"},
		).
		Request(context.Background(), &echo, nil)
	assertNoError(t, "GET with cookies", err)

	cookie := echo.Headers["Cookie"]
	if cookie == "" {
		t.Fatal("expected Cookie header to be set")
	}
	t.Logf("Cookie header: %s", cookie)
}

func Test_StatusError_400(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var apiErr APIResponse
	resp, err := client.New().Get("/error/400").Request(context.Background(), nil, &apiErr)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	// Response metadata is still available even on error
	if resp == nil {
		t.Fatal("expected non-nil Response on error")
	}
	assertEqual(t, "resp.StatusCode", 400, resp.StatusCode)
	assertEqual(t, "resp.Header X-Error-Code", "VALIDATION", resp.Header.Get("X-Error-Code"))

	var statusErr httpspi.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected StatusError, got: %T %v", err, err)
	}
	assertEqual(t, "statusErr.StatusCode", 400, statusErr.StatusCode)
	assertEqual(t, "failure.code", 400, apiErr.Code)
	assertEqual(t, "failure.message", "bad request", apiErr.Message)
}

func Test_StatusError_500(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var apiErr APIResponse
	resp, err := client.New().Get("/error/500").Request(context.Background(), nil, &apiErr)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	assertEqual(t, "resp.StatusCode", 500, resp.StatusCode)

	var statusErr httpspi.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected StatusError, got: %T %v", err, err)
	}
	assertEqual(t, "statusErr.StatusCode", 500, statusErr.StatusCode)
	assertEqual(t, "failure.code", 500, apiErr.Code)
}

func Test_Request_ResponseMetadata(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	resp, err := client.New().Get("/echo").Request(context.Background(), &echo, nil)
	assertNoError(t, "Request", err)

	if resp == nil {
		t.Fatal("expected non-nil Response")
	}
	assertEqual(t, "resp.StatusCode", 200, resp.StatusCode)
	assertEqual(t, "resp.Header Content-Type", "application/json", resp.Header.Get("Content-Type"))
	assertEqual(t, "resp.Header X-Server-Version", "1.0", resp.Header.Get("X-Server-Version"))
	t.Logf("Response: status=%d, headers=%d entries", resp.StatusCode, len(resp.Header))
}

func Test_Receive_ReturnsResponse(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var echo EchoResponse
	resp, err := client.New().Get("/echo").Receive(context.Background(), 5*time.Second, &echo, nil)
	assertNoError(t, "Receive", err)
	defer resp.Body.Close()
	assertEqual(t, "status code", 200, resp.StatusCode)
	assertEqual(t, "method", "GET", echo.Method)
}

func Test_Receive_FailureDecoding(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	var apiErr APIResponse
	resp, err := client.New().Get("/error/400").Receive(context.Background(), 5*time.Second, nil, &apiErr)
	assertNoError(t, "Receive (decode error)", err)
	defer resp.Body.Close()
	assertEqual(t, "status code", 400, resp.StatusCode)
	assertEqual(t, "apiErr.code", 400, apiErr.Code)
}

func Test_Timeout(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := httphelper.NewClient(
		httphelper.WithBaseURL(server.URL),
		httphelper.WithDefaultTimeout(200*time.Millisecond),
	)

	_, err := client.New().Get("/slow").Request(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	t.Logf("Timeout error: %v", err)
}

func Test_New_Isolation(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient(server.URL)

	c1 := client.New().Set("X-Request-ID", "req-1").Get("/echo")
	c2 := client.New().Set("X-Request-ID", "req-2").Get("/echo")

	var echo1, echo2 EchoResponse
	_, err := c1.Request(context.Background(), &echo1, nil)
	assertNoError(t, "c1 request", err)
	_, err = c2.Request(context.Background(), &echo2, nil)
	assertNoError(t, "c2 request", err)

	assertEqual(t, "c1 X-Request-ID", "req-1", echo1.Headers["X-Request-Id"])
	assertEqual(t, "c2 X-Request-ID", "req-2", echo2.Headers["X-Request-Id"])
}

func Test_BaseOverride(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	client := newTestClient("http://will-be-overridden.example.com")

	var echo EchoResponse
	_, err := client.New().Base(server.URL).Get("/echo").Request(context.Background(), &echo, nil)
	assertNoError(t, "Base override", err)
	assertEqual(t, "method", "GET", echo.Method)
}

// ==================== Assertion Helpers ====================

func assertNoError(t *testing.T, label string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("[%s] unexpected error: %v", label, err)
	}
}

func assertEqual[T comparable](t *testing.T, label string, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("[%s] expected %v, got %v", label, expected, actual)
	}
}
