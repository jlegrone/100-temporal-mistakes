package activityhelpers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
)

// rateLimitBackoffMultiplier scales the next retry delay when the upstream
// returns 429 Too Many Requests, so callers slow down without exhausting the
// activity's ScheduleToClose budget too quickly.
const rateLimitBackoffMultiplier = 1.5

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
//   - 408 Request Timeout, 425 Too Early: retryable application error
//   - 429 Too Many Requests: retryable, with a 1.5x multiplier applied to the
//     next retry delay via [temporal.ApplicationErrorOptions.NextRetryDelay]
//   - 4xx codes listed in the non-retryable case below: non-retryable
//     application error (the request is malformed or rejected on its merits)
//   - 5xx and any unrecognized status: retryable application error
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
	case http.StatusTooManyRequests:
		nextDelay := time.Duration(float64(GetNextRetryDelay(ctx)) * rateLimitBackoffMultiplier)
		return temporal.NewApplicationErrorWithOptions(msg, errType, temporal.ApplicationErrorOptions{
			NextRetryDelay: nextDelay,
		})

	case http.StatusRequestTimeout, http.StatusTooEarly:
		return temporal.NewApplicationError(msg, errType)

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

	return temporal.NewApplicationError(msg, errType)
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
