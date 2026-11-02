// Package source defines the contract every data collector implements and
// the shared HTTP client collectors use to reach public sources.
//
// A collector knows only its own source: it fetches, decodes and stores. The
// decoding step of each collector must be a pure function of the payload, with
// no network or database access, so it can be tested by replaying recorded
// payloads from the collector's testdata directory.
package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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

// Retrier is implemented by collectors that choose how soon a failed run is
// retried. Without it the scheduler retries after its default delay (see
// ingest.DueAfter); a weekly collector that downloads large files sets a
// longer delay so an outage is not hammered every few minutes.
type Retrier interface {
	// RetryAfter is how long after a failed run the collector runs again.
	RetryAfter() time.Duration
}

// Pruner is implemented by collectors whose stored data has a retention
// limit. The scheduler calls Prune about hourly.
type Pruner interface {
	Prune(ctx context.Context, now time.Time) error
}

const (
	// maxBody caps a response so a misbehaving source cannot exhaust memory.
	maxBody = 32 << 20
	// doTimeout bounds one Do request unless the request sets its own.
	doTimeout = 60 * time.Second
	// fetchTimeout bounds one Fetch or GetJSON request unless the request
	// sets its own.
	fetchTimeout = 30 * time.Second
)

// Back-off after a source asks us to slow down (HTTP 429, or 503 with
// Retry-After): requests to that host fail fast until the time has passed.
const (
	defaultBackoff = time.Minute
	maxBackoff     = time.Hour
)

// StatusError is a response with a status the caller did not accept.
type StatusError struct {
	Status int
	// RetryAfter is the delay the source asked for, if any.
	RetryAfter time.Duration
}

func (err *StatusError) Error() string {
	if err.RetryAfter > 0 {
		return fmt.Sprintf("unexpected status %d (retry after %s)", err.Status, err.RetryAfter)
	}
	return fmt.Sprintf("unexpected status %d", err.Status)
}

// BackoffError is returned without contacting a host that recently asked us
// to slow down.
type BackoffError struct {
	Host  string
	Until time.Time
}

func (err *BackoffError) Error() string {
	return fmt.Sprintf("backing off %s until %s", err.Host, err.Until.UTC().Format(time.RFC3339))
}

// Client is an HTTP client that identifies the project to source operators
// and backs off from hosts that rate-limit it.
type Client struct {
	http      *http.Client
	userAgent string
	now       func() time.Time

	mutex   sync.Mutex
	backoff map[string]time.Time
}

// NewClient returns a client whose User-Agent names the project and a contact
// (a URL or mailbox) so operators can reach us about our traffic.
func NewClient(contact string) *Client {
	agent := "lanka-weather/0.1"
	if contact != "" {
		agent += " (+" + contact + ")"
	}
	return &Client{
		// Requests carry their own deadline (Request.Timeout), so the client has none.
		http:      &http.Client{},
		userAgent: agent,
		now:       time.Now,
		backoff:   map[string]time.Time{},
	}
}

// UserAgent returns the User-Agent header the client sends.
func (client *Client) UserAgent() string { return client.userAgent }

// Request is one HTTP request to a source.
type Request struct {
	// Method defaults to GET.
	Method string
	URL    string
	// Query is appended to URL.
	Query url.Values
	// Header is sent as given; User-Agent is always the client's own.
	Header http.Header
	// Body is sent as is; Form, when set, is sent instead as an
	// application/x-www-form-urlencoded body.
	Body []byte
	Form url.Values
	// Accept sets the Accept header. Fetch defaults it to application/json.
	Accept string
	// AcceptStatus lists the statuses treated as success; default 200 only.
	AcceptStatus []int
	// MaxBytes caps the response body; it defaults to 32 MiB.
	MaxBytes int64
	// Timeout bounds the whole request, body included. It defaults to 60
	// seconds for Do and 30 seconds for Fetch and GetJSON.
	Timeout time.Duration
}

// Response is a successful response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Fetcher performs requests to sources and returns the body of a successful
// response. *Client implements it; tests replay recorded payloads through it.
type Fetcher interface {
	Fetch(ctx context.Context, request Request) ([]byte, error)
}

// Do sends a request and reads the response body, up to the size limit. A
// status outside AcceptStatus is returned as a *StatusError; a 429, or a 503
// with Retry-After, also makes later requests to the same host fail with a
// *BackoffError until the delay has passed.
func (client *Client) Do(ctx context.Context, request Request) (Response, error) {
	target, err := url.Parse(request.URL)
	if err != nil {
		return Response{}, errors.New("invalid source URL")
	}
	if until, ok := client.backingOff(target.Host); ok {
		return Response{}, &BackoffError{Host: target.Host, Until: until}
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = doTimeout
	}
	limit := request.MaxBytes
	if limit <= 0 {
		limit = maxBody
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	method := request.Method
	if method == "" {
		method = http.MethodGet
	}
	address := request.URL
	if len(request.Query) > 0 {
		separator := "?"
		if strings.Contains(address, "?") {
			separator = "&"
		}
		address += separator + request.Query.Encode()
	}
	var body io.Reader
	switch {
	case request.Form != nil:
		body = strings.NewReader(request.Form.Encode())
	case request.Body != nil:
		body = bytes.NewReader(request.Body)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, method, address, body)
	if err != nil {
		return Response{}, err
	}
	for name, values := range request.Header {
		for _, value := range values {
			httpRequest.Header.Add(name, value)
		}
	}
	httpRequest.Header.Set("User-Agent", client.userAgent)
	if request.Accept != "" {
		httpRequest.Header.Set("Accept", request.Accept)
	}
	if request.Form != nil {
		httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response, err := client.http.Do(httpRequest)
	if err != nil {
		// The error may quote the URL; source URLs never carry credentials.
		return Response{}, err
	}
	defer func() { _ = response.Body.Close() }()
	content, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return Response{}, err
	}
	if int64(len(content)) > limit {
		return Response{}, errors.New("response exceeds size limit")
	}
	accept := request.AcceptStatus
	if len(accept) == 0 {
		accept = []int{http.StatusOK}
	}
	for _, status := range accept {
		if response.StatusCode == status {
			return Response{Status: response.StatusCode, Header: response.Header, Body: content}, nil
		}
	}
	statusErr := &StatusError{Status: response.StatusCode, RetryAfter: retryAfter(response.Header.Get("Retry-After"), client.now())}
	if response.StatusCode == http.StatusTooManyRequests ||
		(response.StatusCode == http.StatusServiceUnavailable && statusErr.RetryAfter > 0) {
		client.backOff(target.Host, statusErr.RetryAfter)
	}
	return Response{Status: response.StatusCode, Header: response.Header, Body: content}, statusErr
}

// Fetch performs a request and returns the body of a successful response.
// Unlike Do it asks for JSON and allows 30 seconds unless the request says
// otherwise.
func (client *Client) Fetch(ctx context.Context, request Request) ([]byte, error) {
	if request.Accept == "" {
		request.Accept = "application/json"
	}
	if request.Timeout <= 0 {
		request.Timeout = fetchTimeout
	}
	response, err := client.Do(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

// GetJSON performs a GET request with query parameters and returns the body.
func (client *Client) GetJSON(ctx context.Context, endpoint string, query url.Values) ([]byte, error) {
	return client.Fetch(ctx, Request{URL: endpoint, Query: query})
}

func (client *Client) backingOff(host string) (time.Time, bool) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	until, ok := client.backoff[host]
	if !ok {
		return time.Time{}, false
	}
	if !client.now().Before(until) {
		delete(client.backoff, host)
		return time.Time{}, false
	}
	return until, true
}

func (client *Client) backOff(host string, delay time.Duration) {
	if delay <= 0 {
		delay = defaultBackoff
	}
	delay = min(delay, maxBackoff)
	client.mutex.Lock()
	defer client.mutex.Unlock()
	client.backoff[host] = client.now().Add(delay)
}

// retryAfter reads a Retry-After header in seconds or as an HTTP date.
func retryAfter(header string, now time.Time) time.Duration {
	if header == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(header); err == nil && at.After(now) {
		return at.Sub(now)
	}
	return 0
}

// IsStatus reports whether err is a *StatusError with one of the statuses.
func IsStatus(err error, statuses ...int) bool {
	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	for _, status := range statuses {
		if statusErr.Status == status {
			return true
		}
	}
	return false
}
