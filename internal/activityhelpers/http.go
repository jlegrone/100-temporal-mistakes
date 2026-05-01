package activityhelpers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
)

// HTTPTransportErrorType is the ApplicationError type used for transport-layer
// failures from [http.Client.Do].
const HTTPTransportErrorType = "Transport"

// HTTPTransportErrorDetails is attached as the structured details payload on
// an ApplicationError produced by [HTTPClient.Do] when the underlying request
// fails at the transport layer. It lets callers branch on the cause without
// parsing error messages.
type HTTPTransportErrorDetails struct {
	// Op is the HTTP operation that failed (e.g. "Post").
	Op string `json:"op"`
	// URL is the destination of the failed request, with any password
	// stripped by net/http.
	URL string `json:"url"`
	// Reason is a short, machine-readable classification of the failure.
	// One of: "dns-not-found", "timeout", "transport".
	Reason string `json:"reason"`
}

// classifyHTTPTransportError converts a transport-layer error from
// [http.Client.Do] into a Temporal ApplicationError. Errors that retries
// cannot fix (an unresolvable hostname) are marked non-retryable; recoverable
// errors (timeouts, connection refused, transient DNS failures) are left
// retryable so the activity's RetryPolicy applies.
//
// In all cases the original error is preserved as the cause and a
// [HTTPTransportErrorDetails] payload is attached so consumers can branch on
// the classification programmatically rather than parsing the message.
func classifyHTTPTransportError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return temporal.NewApplicationErrorWithCause(err.Error(), HTTPTransportErrorType, err)
	}

	details := HTTPTransportErrorDetails{
		Op:     urlErr.Op,
		URL:    urlErr.URL,
		Reason: "transport",
	}

	// NXDOMAIN means the configured endpoint doesn't exist; a retry against
	// the same hostname will keep failing the same way. Transient DNS
	// failures (SERVFAIL, etc.) keep IsNotFound=false and stay retryable.
	var dnsErr *net.DNSError
	nonRetryable := false
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		details.Reason = "dns-not-found"
		nonRetryable = true
	} else if urlErr.Timeout() {
		details.Reason = "timeout"
	}

	return temporal.NewApplicationErrorWithOptions(urlErr.Error(), HTTPTransportErrorType, temporal.ApplicationErrorOptions{
		Cause:        err,
		NonRetryable: nonRetryable,
		Details:      []interface{}{details},
	})
}

// defaultIdempotencyHeader is the header [httpClient] populates from
// [GetIdempotencyToken] when no override is configured. The name follows the
// convention popularized by Stripe; see
// https://docs.stripe.com/api/idempotent_requests.
const defaultIdempotencyHeader = "Idempotency-Key"

// httpClient wraps an *http.Client to add behavior that activities almost
// always want:
//
//   - Transport-layer failures from [http.Client.Do] are translated into
//     structured Temporal ApplicationErrors so callsites don't have to convert
//     *url.Error themselves.
//   - The configured idempotency header is populated from
//     [GetIdempotencyToken] (using the request's context) when the caller
//     hasn't already set it, so retries of the same activity reach the
//     upstream service with the same key.
//
// HTTP responses (any status) are returned to the caller unchanged; pair the
// client with [HTTPResponseError] to map status codes onto retry behavior.
type httpClient struct {
	// Client is the underlying *http.Client. A nil value is treated as
	// [http.DefaultClient].
	Client *http.Client

	// IdempotencyHeader is the request header populated from
	// [GetIdempotencyToken] when the request doesn't already include it.
	// Empty defaults to [defaultIdempotencyHeader] ("Idempotency-Key");
	// override it for upstream services that expect a different name (for
	// example, "X-Idempotency-Key").
	IdempotencyHeader string
}

// DefaultHTTPClient is the package-supplied client backed by
// [http.DefaultClient] using "Idempotency-Key" as its idempotency header.
// Concurrency-safety matches the underlying *http.Client.
var DefaultHTTPClient = &httpClient{
	Client:            http.DefaultClient,
	IdempotencyHeader: defaultIdempotencyHeader,
}

// Do executes req on the underlying client. Before sending, it populates the
// configured idempotency header from [GetIdempotencyToken] using
// req.Context(); if the caller has already populated the header or the
// request was built with a non-activity context (so no token is available),
// the existing value is left untouched. Transport-layer failures are returned
// as Temporal ApplicationErrors (with a [HTTPTransportErrorDetails] payload);
// successful HTTP exchanges are returned to the caller for status-code
// mapping.
func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	header := c.IdempotencyHeader
	if header == "" {
		header = defaultIdempotencyHeader
	}
	if req.Header.Get(header) == "" {
		if token := GetIdempotencyToken(req.Context()); token != "" {
			req.Header.Set(header, token)
		}
	}

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, classifyHTTPTransportError(err)
	}
	return resp, nil
}

// HTTPResponseError translates an HTTP response into a [temporal.ApplicationError]
// suitable for returning from an activity. It returns nil for 2xx and 3xx
// responses; the caller is then responsible for reading and decoding the body.
//
// For non-success responses the helper consumes a 256-byte snippet of the
// response body for the error message; the body is no longer fully readable
// after HTTPResponseError returns an error.
//
// Mapping:
//   - 2xx, 3xx: nil
//   - 4xx codes listed in the non-retryable case below: non-retryable
//     application error (the request is malformed or rejected on its merits)
//   - 408, 425, 429, 5xx, and any unrecognized status: retryable application
//     error
//
// On retryable responses, the server's Retry-After header (RFC 7231 §7.1.3),
// if present in either delta-seconds or HTTP-date form, sets
// [temporal.ApplicationErrorOptions.NextRetryDelay] so the activity respects
// the upstream's hint. When the header is absent, 429 falls back to a
// minimum 3x backoff coefficient via [GetNextRetryDelay] so callers back off
// more aggressively than the policy's default; other retryable codes fall
// back to the policy's natural delay.
//
// Error types are derived from [http.StatusText] with whitespace removed (e.g.,
// "BadRequest", "TooManyRequests", "InternalServerError"). Codes without a
// known reason phrase fall back to "HTTP<code>".
func HTTPResponseError(ctx context.Context, resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	msg := fmt.Sprintf("%s: %s", resp.Status, httpBodySnippet(resp))
	errType := httpErrorType(resp.StatusCode)

	switch resp.StatusCode {
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusPaymentRequired,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusNotAcceptable,
		http.StatusProxyAuthRequired,
		http.StatusConflict,
		http.StatusGone,
		http.StatusLengthRequired,
		http.StatusPreconditionFailed,
		http.StatusRequestEntityTooLarge,
		http.StatusRequestURITooLong,
		http.StatusUnsupportedMediaType,
		http.StatusRequestedRangeNotSatisfiable,
		http.StatusExpectationFailed,
		http.StatusTeapot,
		http.StatusMisdirectedRequest,
		http.StatusUnprocessableEntity,
		http.StatusLocked,
		http.StatusFailedDependency,
		http.StatusUpgradeRequired,
		http.StatusPreconditionRequired,
		http.StatusRequestHeaderFieldsTooLarge,
		http.StatusUnavailableForLegalReasons:
		return temporal.NewNonRetryableApplicationError(msg, errType, nil)
	}

	// Retryable from here. Honor the server's Retry-After hint when
	// present; fall back to an aggressive policy floor for 429.
	delay := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	if delay == 0 && resp.StatusCode == http.StatusTooManyRequests {
		delay = GetNextRetryDelay(ctx, 3)
	}
	return temporal.NewApplicationErrorWithOptions(msg, errType, temporal.ApplicationErrorOptions{
		NextRetryDelay: delay,
	})
}

// parseRetryAfter interprets the value of an HTTP Retry-After header per
// RFC 7231 §7.1.3, accepting either a non-negative integer of seconds or an
// HTTP-date. It returns 0 for empty, malformed, or past-dated values; the
// caller should then fall back to its default backoff.
func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if secs, err := strconv.ParseUint(value, 10, 32); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}

func httpErrorType(status int) string {
	text := http.StatusText(status)
	if text == "" {
		return fmt.Sprintf("http_%d", status)
	}
	return strings.ReplaceAll(text, " ", "")
}

func httpBodySnippet(resp *http.Response) string {
	if resp.Body == nil {
		return ""
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return string(bytes.TrimSpace(b))
}
