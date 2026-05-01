package activityhelpers

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestHTTPResponseError(t *testing.T) {
	cases := map[string]struct {
		status       int
		wantNil      bool
		wantNonRetry bool
		wantType     string
	}{
		"200 OK is nil":                        {status: http.StatusOK, wantNil: true},
		"204 NoContent is nil":                 {status: http.StatusNoContent, wantNil: true},
		"301 MovedPermanently is nil":          {status: http.StatusMovedPermanently, wantNil: true},
		"400 BadRequest is non-retryable":      {status: http.StatusBadRequest, wantNonRetry: true, wantType: "BadRequest"},
		"401 Unauthorized is non-retryable":    {status: http.StatusUnauthorized, wantNonRetry: true, wantType: "Unauthorized"},
		"404 NotFound is non-retryable":        {status: http.StatusNotFound, wantNonRetry: true, wantType: "NotFound"},
		"408 RequestTimeout is retryable":      {status: http.StatusRequestTimeout, wantType: "RequestTimeout"},
		"425 TooEarly is retryable":            {status: http.StatusTooEarly, wantType: "TooEarly"},
		"429 TooManyRequests is retryable":     {status: http.StatusTooManyRequests, wantType: "TooManyRequests"},
		"500 InternalServerError is retryable": {status: http.StatusInternalServerError, wantType: "InternalServerError"},
		"503 ServiceUnavailable is retryable":  {status: http.StatusServiceUnavailable, wantType: "ServiceUnavailable"},
		"unlisted 4xx defaults to retryable":   {status: 419, wantType: "http_419"},
		"unknown status falls back to HTTPnnn": {status: 599, wantType: "http_599"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp := newResponse(tc.status, "body snippet")
			err := HTTPResponseError(context.Background(), resp)
			if tc.wantNil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)

			var appErr *temporal.ApplicationError
			require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
			require.Equal(t, tc.wantType, appErr.Type())
			require.Equal(t, tc.wantNonRetry, appErr.NonRetryable())
		})
	}
}

func TestHTTPResponseError_IncludesBodySnippet(t *testing.T) {
	resp := newResponse(http.StatusBadRequest, "missing currency field")
	err := HTTPResponseError(context.Background(), resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing currency field")
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cases := map[string]struct {
		value string
		want  time.Duration
	}{
		"empty":                       {value: "", want: 0},
		"whitespace":                  {value: "   ", want: 0},
		"zero seconds":                {value: "0", want: 0},
		"thirty seconds":              {value: "30", want: 30 * time.Second},
		"large seconds":               {value: "3600", want: time.Hour},
		"http-date in future":         {value: now.Add(45 * time.Second).UTC().Format(http.TimeFormat), want: 45 * time.Second},
		"http-date in past":           {value: now.Add(-time.Minute).UTC().Format(http.TimeFormat), want: 0},
		"malformed":                   {value: "tomorrow", want: 0},
		"negative integer is invalid": {value: "-5", want: 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseRetryAfter(tc.value, now))
		})
	}
}

func TestHTTPResponseError_RetryAfterHeader(t *testing.T) {
	cases := map[string]struct {
		status       int
		retryAfter   string
		wantDelayMin time.Duration
		wantDelayMax time.Duration
	}{
		"429 honors integer Retry-After": {
			status:       http.StatusTooManyRequests,
			retryAfter:   "45",
			wantDelayMin: 45 * time.Second,
			wantDelayMax: 45 * time.Second,
		},
		"503 honors integer Retry-After": {
			status:       http.StatusServiceUnavailable,
			retryAfter:   "10",
			wantDelayMin: 10 * time.Second,
			wantDelayMax: 10 * time.Second,
		},
		"408 honors integer Retry-After": {
			status:       http.StatusRequestTimeout,
			retryAfter:   "5",
			wantDelayMin: 5 * time.Second,
			wantDelayMax: 5 * time.Second,
		},
		"500 with HTTP-date in future": {
			status:       http.StatusInternalServerError,
			retryAfter:   time.Now().UTC().Add(20 * time.Second).Format(http.TimeFormat),
			wantDelayMin: 18 * time.Second, // allow ~2s slack for test timing
			wantDelayMax: 20 * time.Second,
		},
		"503 without Retry-After leaves delay unset": {
			status:       http.StatusServiceUnavailable,
			retryAfter:   "",
			wantDelayMin: 0,
			wantDelayMax: 0,
		},
		"429 without Retry-After falls back outside activity ctx (delay=0)": {
			// GetNextRetryDelay returns 0 outside an activity context, so
			// the test asserts we still emit a retryable ApplicationError.
			status:       http.StatusTooManyRequests,
			retryAfter:   "",
			wantDelayMin: 0,
			wantDelayMax: 0,
		},
		"429 ignores malformed Retry-After (falls back)": {
			status:       http.StatusTooManyRequests,
			retryAfter:   "not-a-thing",
			wantDelayMin: 0,
			wantDelayMax: 0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp := newResponse(tc.status, "")
			if tc.retryAfter != "" {
				resp.Header.Set("Retry-After", tc.retryAfter)
			}
			err := HTTPResponseError(context.Background(), resp)
			require.Error(t, err)
			var appErr *temporal.ApplicationError
			require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
			assert.False(t, appErr.NonRetryable(), "must remain retryable")
			delay := appErr.NextRetryDelay()
			assert.GreaterOrEqual(t, delay, tc.wantDelayMin, "delay below floor")
			assert.LessOrEqual(t, delay, tc.wantDelayMax, "delay above ceiling")
		})
	}
}

// timeoutError implements net.Error and reports Timeout() == true. It lets us
// drive the timeout branch of classifyHTTPTransportError without spinning up a
// flaky real-network test.
type timeoutError struct{}

func (timeoutError) Error() string { return "i/o timeout" }
func (timeoutError) Timeout() bool { return true }

func TestClassifyHTTPTransportError_DNSNotFoundIsNonRetryable(t *testing.T) {
	dnsErr := &net.DNSError{
		Err:        "no such host",
		Name:       "missing.example",
		IsNotFound: true,
	}
	urlErr := &url.Error{
		Op:  "Post",
		URL: "https://missing.example/v1/charges",
		Err: dnsErr,
	}

	err := classifyHTTPTransportError(urlErr)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.Equal(t, HTTPTransportErrorType, appErr.Type())
	require.True(t, appErr.NonRetryable(), "DNS NXDOMAIN must be non-retryable")

	var details HTTPTransportErrorDetails
	require.NoError(t, appErr.Details(&details))
	require.Equal(t, "dns-not-found", details.Reason)
	require.Equal(t, "Post", details.Op)
	require.Equal(t, "https://missing.example/v1/charges", details.URL)
}

func TestClassifyHTTPTransportError_TimeoutIsRetryable(t *testing.T) {
	urlErr := &url.Error{
		Op:  "Post",
		URL: "https://api.example.com/v1/charges",
		Err: timeoutError{},
	}

	err := classifyHTTPTransportError(urlErr)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.Equal(t, HTTPTransportErrorType, appErr.Type())
	require.False(t, appErr.NonRetryable(), "timeouts must remain retryable")

	var details HTTPTransportErrorDetails
	require.NoError(t, appErr.Details(&details))
	require.Equal(t, "timeout", details.Reason)
}

func TestClassifyHTTPTransportError_GenericIsRetryable(t *testing.T) {
	urlErr := &url.Error{
		Op:  "Post",
		URL: "https://api.example.com/v1/charges",
		Err: errors.New("connection reset by peer"),
	}

	err := classifyHTTPTransportError(urlErr)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.Equal(t, HTTPTransportErrorType, appErr.Type())
	require.False(t, appErr.NonRetryable(), "generic transport errors must remain retryable")

	var details HTTPTransportErrorDetails
	require.NoError(t, appErr.Details(&details))
	require.Equal(t, "transport", details.Reason)
}

func TestClassifyHTTPTransportError_NonURLErrorFallback(t *testing.T) {
	// When the error isn't a *url.Error (unlikely from http.Client.Do but
	// possible from custom transports), fall back to a plain ApplicationError
	// without details.
	err := classifyHTTPTransportError(errors.New("boom"))

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.Equal(t, HTTPTransportErrorType, appErr.Type())
	require.False(t, appErr.NonRetryable())

	var details HTTPTransportErrorDetails
	require.Error(t, appErr.Details(&details), "expected no details payload on fallback path")
}

func TestHTTPClient_DoConnectionRefused(t *testing.T) {
	// Bind a port, capture its URL, then close the listener so subsequent
	// connect attempts get ECONNREFUSED.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := srv.URL
	srv.Close()

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	require.NoError(t, err)

	_, err = DefaultHTTPClient.Do(req)
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.Equal(t, HTTPTransportErrorType, appErr.Type())
	require.False(t, appErr.NonRetryable(), "connection refused must remain retryable")

	var details HTTPTransportErrorDetails
	require.NoError(t, appErr.Details(&details))
	require.NotEmpty(t, details.URL)
	require.Contains(t, []string{"transport", "timeout"}, details.Reason)
}

func TestHTTPClient_DoNilClientUsesDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := &httpClient{}
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	resp, err := c.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// captureHeaderServer returns an httptest.Server that records the value of
// the named header from the most recent request and replies with 200 OK.
func captureHeaderServer(t *testing.T, header string, captured *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured = r.Header.Get(header)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHTTPClient_DoSetsIdempotencyKey(t *testing.T) {
	var seen string
	srv := captureHeaderServer(t, defaultIdempotencyHeader, &seen)

	act := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, nil)
		if err != nil {
			return err
		}
		resp, err := DefaultHTTPClient.Do(req)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(act)
	_, err := env.ExecuteActivity(act)
	require.NoError(t, err)

	require.NotEmpty(t, seen, "expected client to populate Idempotency-Key from activity context")
}

func TestHTTPClient_DoUsesConfiguredHeader(t *testing.T) {
	const customHeader = "X-Idempotency-Key"
	var seen, defaultSeen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get(customHeader)
		defaultSeen = r.Header.Get(defaultIdempotencyHeader)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := &httpClient{IdempotencyHeader: customHeader}
	act := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(act)
	_, err := env.ExecuteActivity(act)
	require.NoError(t, err)

	require.NotEmpty(t, seen, "expected client to populate the configured header")
	require.Empty(t, defaultSeen, "client must not also populate the default header when overridden")
}

func TestHTTPClient_DoPreservesExistingIdempotencyKey(t *testing.T) {
	var seen string
	srv := captureHeaderServer(t, defaultIdempotencyHeader, &seen)

	const explicit = "caller-supplied-key"
	act := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, nil)
		if err != nil {
			return err
		}
		req.Header.Set(defaultIdempotencyHeader, explicit)
		resp, err := DefaultHTTPClient.Do(req)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(act)
	_, err := env.ExecuteActivity(act)
	require.NoError(t, err)

	require.Equal(t, explicit, seen, "client must not overwrite a caller-supplied Idempotency-Key")
}

func TestHTTPClient_DoLeavesHeaderUnsetOutsideActivity(t *testing.T) {
	var seen string
	srv := captureHeaderServer(t, defaultIdempotencyHeader, &seen)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, nil)
	require.NoError(t, err)

	resp, err := DefaultHTTPClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	require.Empty(t, seen, "no activity context means no token to populate")
}

func TestHTTPClient_DoZeroValueFallsBackToDefaultHeader(t *testing.T) {
	var seen string
	srv := captureHeaderServer(t, defaultIdempotencyHeader, &seen)

	client := &httpClient{} // zero value: empty IdempotencyHeader
	act := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(act)
	_, err := env.ExecuteActivity(act)
	require.NoError(t, err)

	require.NotEmpty(t, seen, "zero-value HTTPClient must default to %q", defaultIdempotencyHeader)
}
