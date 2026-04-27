package activityhelpers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
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
