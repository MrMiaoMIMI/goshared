// Package httpsp provides the concrete implementation of httpspi.Client
// backed by the standard net/http package.
package httpsp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MrMiaoMIMI/goshared/http/httpspi"
)

var _ httpspi.Client = (*httpClient)(nil)

// ClientOption configures the httpClient at construction time.
type ClientOption func(*clientConfig)

type clientConfig struct {
	httpClient     *http.Client
	baseURL        string
	defaultTimeout time.Duration
	defaultHeaders map[string]string
	maxRetries     int
	retryDelay     time.Duration
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		httpClient:     &http.Client{},
		defaultTimeout: 30 * time.Second,
	}
}

// WithHTTPClient sets the underlying *http.Client used for sending requests.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(cfg *clientConfig) { cfg.httpClient = c }
}

// WithBaseURL sets the base URL for all requests.
func WithBaseURL(baseURL string) ClientOption {
	return func(cfg *clientConfig) { cfg.baseURL = baseURL }
}

// WithDefaultTimeout sets the default timeout applied by Request().
// Receive() uses its own explicit timeout parameter instead.
func WithDefaultTimeout(d time.Duration) ClientOption {
	return func(cfg *clientConfig) { cfg.defaultTimeout = d }
}

// WithDefaultHeaders sets headers that are applied to every request.
func WithDefaultHeaders(headers map[string]string) ClientOption {
	return func(cfg *clientConfig) { cfg.defaultHeaders = headers }
}

// WithRetry sets the max retry count and delay between retries for transient failures.
// Only network errors and 5xx responses trigger retries.
// Delay must be positive; enforces a minimum of 100ms to prevent tight loops.
// maxRetries must be >= 0; negative values are clamped to 0.
func WithRetry(maxRetries int, delay time.Duration) ClientOption {
	return func(cfg *clientConfig) {
		if maxRetries < 0 {
			maxRetries = 0
		}
		cfg.maxRetries = maxRetries
		if delay < 100*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		cfg.retryDelay = delay
	}
}

type multipartData struct {
	fieldName   string
	fileName    string
	fileData    []byte
	extraFields map[string]string
}

type httpClient struct {
	rawClient      *http.Client
	baseURL        string
	method         string
	pathURL        string
	header         http.Header
	queryStructs   []any
	queryParams    url.Values
	headerStructs  []any
	bodyJSON       any
	bodyXML        any
	bodyForm       map[string]string
	bodyMultipart  *multipartData
	bodyBytes      []byte
	respDecoder    httpspi.ResponseDecoder
	defaultTimeout time.Duration
	maxRetries     int
	retryDelay     time.Duration
	cookies        []*http.Cookie
}

// NewHTTPClient creates a new Client with the given options.
func NewHTTPClient(opts ...ClientOption) httpspi.Client {
	cfg := defaultClientConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	header := make(http.Header)
	for k, v := range cfg.defaultHeaders {
		header.Set(k, v)
	}

	return &httpClient{
		rawClient:      cfg.httpClient,
		baseURL:        cfg.baseURL,
		header:         header,
		defaultTimeout: cfg.defaultTimeout,
		maxRetries:     cfg.maxRetries,
		retryDelay:     cfg.retryDelay,
		respDecoder:    &jsonDecoder{},
	}
}

func (c *httpClient) New() httpspi.Client {
	return &httpClient{
		rawClient:      c.rawClient,
		baseURL:        c.baseURL,
		header:         c.header.Clone(),
		respDecoder:    c.respDecoder,
		defaultTimeout: c.defaultTimeout,
		maxRetries:     c.maxRetries,
		retryDelay:     c.retryDelay,
	}
}

// ==================== Header ====================

func (c *httpClient) Add(key, value string) httpspi.Client {
	c.header.Add(key, value)
	return c
}

func (c *httpClient) Set(key, value string) httpspi.Client {
	c.header.Set(key, value)
	return c
}

func (c *httpClient) Header(header http.Header) httpspi.Client {
	for k, values := range header {
		c.header[k] = values
	}
	return c
}

func (c *httpClient) HeaderMap(headers map[string]string) httpspi.Client {
	for k, v := range headers {
		c.header.Set(k, v)
	}
	return c
}

func (c *httpClient) BearerToken(token string) httpspi.Client {
	c.header.Set("Authorization", "Bearer "+token)
	return c
}

func (c *httpClient) BasicAuth(username, password string) httpspi.Client {
	cred := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	c.header.Set("Authorization", "Basic "+cred)
	return c
}

func (c *httpClient) AddCookies(cookies ...*http.Cookie) httpspi.Client {
	c.cookies = append(c.cookies, cookies...)
	return c
}

func (c *httpClient) HeaderStruct(headerStruct any) httpspi.Client {
	if headerStruct != nil {
		c.headerStructs = append(c.headerStructs, headerStruct)
	}
	return c
}

// ==================== URL ====================

func (c *httpClient) Base(rawURL string) httpspi.Client {
	c.baseURL = rawURL
	return c
}

func (c *httpClient) QueryStruct(queryStruct any) httpspi.Client {
	if queryStruct != nil {
		c.queryStructs = append(c.queryStructs, queryStruct)
	}
	return c
}

func (c *httpClient) QueryParam(key, value string) httpspi.Client {
	if c.queryParams == nil {
		c.queryParams = make(url.Values)
	}
	c.queryParams.Add(key, value)
	return c
}

// ==================== Composite ====================

func (c *httpClient) RequestStruct(requestStruct any) httpspi.Client {
	if requestStruct == nil {
		return c
	}
	c.QueryStruct(requestStruct)
	c.HeaderStruct(requestStruct)

	jsonBody, err := extractJSONBody(requestStruct)
	if err == nil && jsonBody != nil {
		c.bodyJSON = jsonBody
		c.header.Set("Content-Type", "application/json")
	}
	return c
}

// ==================== Method ====================

func (c *httpClient) Get(pathURL string) httpspi.Client {
	c.method = http.MethodGet
	c.pathURL = pathURL
	return c
}

func (c *httpClient) Post(pathURL string) httpspi.Client {
	c.method = http.MethodPost
	c.pathURL = pathURL
	return c
}

func (c *httpClient) Head(pathURL string) httpspi.Client {
	c.method = http.MethodHead
	c.pathURL = pathURL
	return c
}

func (c *httpClient) Put(pathURL string) httpspi.Client {
	c.method = http.MethodPut
	c.pathURL = pathURL
	return c
}

func (c *httpClient) Patch(pathURL string) httpspi.Client {
	c.method = http.MethodPatch
	c.pathURL = pathURL
	return c
}

func (c *httpClient) Delete(pathURL string) httpspi.Client {
	c.method = http.MethodDelete
	c.pathURL = pathURL
	return c
}

func (c *httpClient) Options(pathURL string) httpspi.Client {
	c.method = http.MethodOptions
	c.pathURL = pathURL
	return c
}

// ==================== Body ====================

func (c *httpClient) BodyJSON(bodyJSON any) httpspi.Client {
	if bodyJSON != nil {
		c.bodyJSON = bodyJSON
		c.header.Set("Content-Type", "application/json")
	}
	return c
}

func (c *httpClient) BodyForm(values map[string]string) httpspi.Client {
	if values != nil {
		c.bodyForm = values
		c.header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return c
}

func (c *httpClient) BodyXML(bodyXML any) httpspi.Client {
	if bodyXML != nil {
		c.bodyXML = bodyXML
		c.header.Set("Content-Type", "application/xml")
	}
	return c
}

func (c *httpClient) BodyMultipart(fieldName, fileName string, fileData []byte, extraFields map[string]string) httpspi.Client {
	c.bodyMultipart = &multipartData{
		fieldName:   fieldName,
		fileName:    fileName,
		fileData:    fileData,
		extraFields: extraFields,
	}
	return c
}

func (c *httpClient) BodyBytes(b []byte) httpspi.Client {
	c.bodyBytes = b
	return c
}

// ==================== Send ====================

func (c *httpClient) ResponseDecoder(decoder httpspi.ResponseDecoder) httpspi.Client {
	c.respDecoder = decoder
	return c
}

func (c *httpClient) Receive(ctx context.Context, timeout time.Duration, successV, failureV any) (*http.Response, error) {
	reqURL, err := c.buildURL()
	if err != nil {
		return nil, err
	}

	if err := c.applyHeaderStructs(); err != nil {
		return nil, err
	}

	method := c.method
	if method == "" {
		method = http.MethodGet
	}

	maxAttempts := 1 + c.maxRetries
	var lastResp *http.Response
	var lastErr error

	for attempt := range maxAttempts {
		body, err := c.buildBody()
		if err != nil {
			return nil, err
		}

		reqCtx := ctx
		var cancel context.CancelFunc
		if timeout > 0 {
			reqCtx, cancel = context.WithTimeout(ctx, timeout)
		}

		req, err := http.NewRequestWithContext(reqCtx, method, reqURL, body)
		if err != nil {
			if cancel != nil {
				cancel()
			}
			return nil, err
		}

		req.Header = c.header.Clone()
		for _, cookie := range c.cookies {
			req.AddCookie(cookie)
		}

		resp, err := c.rawClient.Do(req)
		if cancel != nil {
			cancel()
		}

		if err != nil {
			lastErr = err
			if attempt < maxAttempts-1 && c.retryDelay > 0 {
				time.Sleep(c.retryDelay)
			}
			continue
		}

		if resp.StatusCode >= 500 && attempt < maxAttempts-1 {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = httpspi.StatusError{StatusCode: resp.StatusCode, Status: resp.Status}
			if c.retryDelay > 0 {
				time.Sleep(c.retryDelay)
			}
			continue
		}

		decoder := c.respDecoder
		if decoder == nil {
			decoder = &jsonDecoder{}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if successV != nil {
				err = decoder.Decode(resp, successV)
			}
		} else {
			if failureV != nil {
				err = decoder.Decode(resp, failureV)
			}
		}

		return resp, err
	}

	return lastResp, lastErr
}

func (c *httpClient) Request(ctx context.Context, successV, failureV any) (*httpspi.Response, error) {
	resp, err := c.Receive(ctx, c.defaultTimeout, successV, failureV)
	if resp != nil && resp.Body != nil {
		defer func() {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}

	var result *httpspi.Response
	if resp != nil {
		result = &httpspi.Response{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Header:     resp.Header,
		}
	}

	if err != nil {
		return result, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, httpspi.StatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	return result, nil
}

// ==================== Internal helpers ====================

func (c *httpClient) applyHeaderStructs() error {
	for _, hs := range c.headerStructs {
		encoded, err := encodeHeaderStruct(hs)
		if err != nil {
			return fmt.Errorf("http: failed to encode header struct: %w", err)
		}
		for k, vs := range encoded {
			for _, v := range vs {
				c.header.Set(k, v)
			}
		}
	}
	c.headerStructs = nil
	return nil
}

func (c *httpClient) buildURL() (string, error) {
	baseURL := c.baseURL
	pathURL := c.pathURL

	var rawURL string
	switch {
	case pathURL != "" && baseURL != "":
		rawURL = strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(pathURL, "/")
	case pathURL != "":
		rawURL = pathURL
	case baseURL != "":
		rawURL = baseURL
	default:
		return "", fmt.Errorf("http: URL is empty, set Base or a path via method")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("http: invalid URL %q: %w", rawURL, err)
	}

	q := parsedURL.Query()

	for _, qs := range c.queryStructs {
		values, encErr := encodeQueryStruct(qs)
		if encErr != nil {
			return "", fmt.Errorf("http: failed to encode query struct: %w", encErr)
		}
		for k, vs := range values {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
	}

	for k, vs := range c.queryParams {
		for _, v := range vs {
			q.Add(k, v)
		}
	}

	if len(q) > 0 {
		parsedURL.RawQuery = q.Encode()
	}

	return parsedURL.String(), nil
}

func (c *httpClient) buildBody() (io.Reader, error) {
	if c.bodyJSON != nil {
		buf, err := json.Marshal(c.bodyJSON)
		if err != nil {
			return nil, fmt.Errorf("http: failed to marshal JSON body: %w", err)
		}
		return bytes.NewReader(buf), nil
	}
	if c.bodyXML != nil {
		buf, err := xml.Marshal(c.bodyXML)
		if err != nil {
			return nil, fmt.Errorf("http: failed to marshal XML body: %w", err)
		}
		return bytes.NewReader(buf), nil
	}
	if c.bodyForm != nil {
		form := url.Values{}
		for k, v := range c.bodyForm {
			form.Set(k, v)
		}
		return strings.NewReader(form.Encode()), nil
	}
	if c.bodyMultipart != nil {
		return c.buildMultipartBody()
	}
	if len(c.bodyBytes) > 0 {
		return bytes.NewReader(c.bodyBytes), nil
	}
	return nil, nil
}

func (c *httpClient) buildMultipartBody() (io.Reader, error) {
	mp := c.bodyMultipart
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile(mp.fieldName, mp.fileName)
	if err != nil {
		return nil, fmt.Errorf("http: failed to create form file: %w", err)
	}
	if _, err := part.Write(mp.fileData); err != nil {
		return nil, fmt.Errorf("http: failed to write file data: %w", err)
	}
	for k, v := range mp.extraFields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("http: failed to write field %q: %w", k, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("http: failed to close multipart writer: %w", err)
	}

	c.header.Set("Content-Type", writer.FormDataContentType())
	return &buf, nil
}
