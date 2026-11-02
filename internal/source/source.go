// Package source defines the contract every data collector implements and
// the shared HTTP client collectors use to reach public sources.
//
// A collector knows only its own source: it fetches, decodes and stores. The
// decoding step of each collector must be a pure function of the payload, with
// no network or database access, so it can be tested by replaying recorded
// payloads from the collector's testdata directory.
package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Collector fetches one data source and stores what it finds.
type Collector interface {
	// Name is the source's stable identifier, as listed in docs/data-sources.md.
	Name() string
	// Interval is how often the source is fetched.
	Interval() time.Duration
	// Collect fetches and stores new data, returning the number of records stored.
	Collect(ctx context.Context, now time.Time) (int, error)
}

// Retrier is implemented by collectors whose interval is long, so that a
// failed run is retried sooner than a full interval later.
type Retrier interface {
	// RetryAfter is how long after a failed run the collector runs again.
	RetryAfter() time.Duration
}

const (
	// maxBody caps a response so a misbehaving source cannot exhaust memory.
	maxBody = 32 << 20
	// defaultTimeout bounds one request unless the request sets its own.
	defaultTimeout = 30 * time.Second
)

// Client is an HTTP client that identifies the project to source operators.
type Client struct {
	http      *http.Client
	userAgent string
}

// NewClient returns a client whose User-Agent names the project and a contact
// (a URL or mailbox) so operators can reach us about our traffic.
func NewClient(contact string) *Client {
	agent := "lanka-weather/0.1"
	if contact != "" {
		agent += " (+" + contact + ")"
	}
	// Requests carry their own deadline (Request.Timeout), so the client has none.
	return &Client{http: &http.Client{}, userAgent: agent}
}

// UserAgent returns the User-Agent header the client sends.
func (client *Client) UserAgent() string { return client.userAgent }

// Request describes one request to a source.
type Request struct {
	// Method defaults to GET.
	Method string
	URL    string
	Query  url.Values
	// Form, when set, is sent as an application/x-www-form-urlencoded body.
	Form url.Values
	// Accept defaults to application/json.
	Accept string
	// MaxBytes caps the response body; it defaults to 32 MiB.
	MaxBytes int64
	// Timeout bounds the whole request; it defaults to 30 seconds.
	Timeout time.Duration
}

// Fetcher performs requests to sources. *Client implements it; tests replay
// recorded payloads through it.
type Fetcher interface {
	Fetch(ctx context.Context, request Request) ([]byte, error)
}

// Fetch performs a request and returns the body of a 200 response.
func (client *Client) Fetch(ctx context.Context, spec Request) ([]byte, error) {
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	limit := spec.MaxBytes
	if limit <= 0 {
		limit = maxBody
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	method := spec.Method
	if method == "" {
		method = http.MethodGet
	}
	target := spec.URL
	if len(spec.Query) > 0 {
		target += "?" + spec.Query.Encode()
	}
	var body io.Reader
	if spec.Form != nil {
		body = strings.NewReader(spec.Form.Encode())
	}
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	accept := spec.Accept
	if accept == "" {
		accept = "application/json"
	}
	request.Header.Set("User-Agent", client.userAgent)
	request.Header.Set("Accept", accept)
	if spec.Form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	payload, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > limit {
		return nil, errors.New("response exceeds size limit")
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	return payload, nil
}

// GetJSON performs a GET request with query parameters and returns the body.
func (client *Client) GetJSON(ctx context.Context, endpoint string, query url.Values) ([]byte, error) {
	return client.Fetch(ctx, Request{URL: endpoint, Query: query})
}
