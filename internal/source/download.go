package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// LongRunner is implemented by collectors whose runs take minutes, such as
// those that download large archives. The scheduler runs them apart from the
// others, so a slow download never delays a quarter-hourly source, and gives
// them this long instead of its usual limit.
type LongRunner interface {
	LongRunTimeout() time.Duration
}

// Download streams a response to a file instead of memory, for archives above
// the usual body limit. MaxBytes is required: the body is cut off and the
// download fails beyond it. The request's Timeout bounds the whole transfer,
// and must be set. The file is written at path, which is removed on failure.
func (client *Client) Download(ctx context.Context, request Request, path string) (http.Header, int64, error) {
	if request.MaxBytes <= 0 || request.Timeout <= 0 {
		return nil, 0, errors.New("a download needs MaxBytes and Timeout")
	}
	target, err := url.Parse(request.URL)
	if err != nil {
		return nil, 0, errors.New("invalid source URL")
	}
	if until, ok := client.backingOff(target.Host); ok {
		return nil, 0, &BackoffError{Host: target.Host, Until: until}
	}
	ctx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	address := request.URL
	if len(request.Query) > 0 {
		separator := "?"
		if strings.Contains(address, "?") {
			separator = "&"
		}
		address += separator + request.Query.Encode()
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, 0, err
	}
	httpRequest.Header.Set("User-Agent", client.userAgent)
	if request.Accept != "" {
		httpRequest.Header.Set("Accept", request.Accept)
	}
	response, err := client.http.Do(httpRequest)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		statusErr := &StatusError{Status: response.StatusCode, RetryAfter: retryAfter(response.Header.Get("Retry-After"), client.now())}
		if response.StatusCode == http.StatusTooManyRequests ||
			(response.StatusCode == http.StatusServiceUnavailable && statusErr.RetryAfter > 0) {
			client.backOff(target.Host, statusErr.RetryAfter)
		}
		return response.Header, 0, statusErr
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, 0, err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, request.MaxBytes+1))
	closeErr := file.Close()
	switch {
	case copyErr != nil:
		_ = os.Remove(path)
		return nil, 0, fmt.Errorf("download body: %w", copyErr)
	case closeErr != nil:
		_ = os.Remove(path)
		return nil, 0, closeErr
	case written > request.MaxBytes:
		_ = os.Remove(path)
		return nil, 0, fmt.Errorf("download exceeds %d bytes", request.MaxBytes)
	}
	return response.Header, written, nil
}
