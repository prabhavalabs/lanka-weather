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

// maxBody caps a response so a misbehaving source cannot exhaust memory.
const maxBody = 32 << 20

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
	return &Client{http: &http.Client{Timeout: 30 * time.Second}, userAgent: agent}
}

// UserAgent returns the User-Agent header the client sends.
func (client *Client) UserAgent() string { return client.userAgent }

// GetJSON performs a GET request with query parameters and returns the body.
func (client *Client) GetJSON(ctx context.Context, endpoint string, query url.Values) ([]byte, error) {
	target := endpoint
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", client.userAgent)
	request.Header.Set("Accept", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, errors.New("response exceeds size limit")
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	return body, nil
}
