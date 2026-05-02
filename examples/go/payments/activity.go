// Package payments contains the final-state code examples for the "Activities"
// chapter of SLIDES.md. The ChargePayment activity demonstrates handling
// downstream service errors, mapping HTTP status codes to retry behavior, and
// passing a deterministic idempotency key to the upstream payments API.
package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.temporal.io/sdk/temporal"

	"github.com/jlegrone/100-temporal-mistakes/internal/activityhelpers"
)

// ChargePaymentRequest models a charge in the upstream payments API.
type ChargePaymentRequest struct {
	CustomerID  string `json:"customer_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

// ChargePaymentResponse is the subset of the upstream response the workflow
// cares about; see "Activities: Avoid Passing Too Much Information".
type ChargePaymentResponse struct {
	ChargeID string `json:"id"`
	Status   string `json:"status"`
}

// Worker groups payments activities so they can share configuration.
type Worker struct {
	endpoint string
}

// NewWorker constructs a Worker that posts charges to endpoint.
func NewWorker(endpoint string) *Worker {
	return &Worker{endpoint: endpoint}
}

// ChargePayment posts a charge to the payments API. Status-code-to-retry
// classification is delegated to [activityhelpers.HTTPResponseError]:
//
//   - 2xx, 3xx: success
//   - 4xx (400, 401, 403, 404, ...): non-retryable
//   - 408, 425, 429, 5xx, unknown: retryable; honors the server's
//     Retry-After header when present and applies a 3x backoff floor
//     for 429 when it isn't
//
// The Idempotency-Key header is derived from the workflow + activity identity
// so retries are coalesced by the upstream service.
func (w *Worker) ChargePayment(ctx context.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
	httpReq, err := w.buildChargeRequest(ctx, req)
	if err != nil {
		// Errors returned by buildChargeRequest are determinstic, so don't retry them.
		return nil, temporal.NewApplicationErrorWithOptions(err.Error(), "BuildRequest", temporal.ApplicationErrorOptions{
			NonRetryable: true,
			Cause:        err,
		})
	}

	resp, err := activityhelpers.DefaultHTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := activityhelpers.HTTPResponseError(ctx, resp); err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {

	}

	var out ChargePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		// A type mismatch between the JSON value and the Go field is a
		// schema disagreement that won't resolve on retry; everything else
		// (syntax errors from a truncated body, I/O errors mid-stream) can
		// reasonably be retried.
		var typeErr *json.UnmarshalTypeError
		return nil, temporal.NewApplicationErrorWithOptions(
			fmt.Sprintf("decode response: %v", err),
			"DecodeResponse",
			temporal.ApplicationErrorOptions{
				NonRetryable: errors.As(err, &typeErr),
				Cause:        err,
			},
		)
	}
	return &out, nil
}

// buildChargeRequest serializes a ChargePaymentRequest as JSON and builds the
// HTTP request.
func (w *Worker) buildChargeRequest(ctx context.Context, req ChargePaymentRequest) (*http.Request, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq, nil
}
