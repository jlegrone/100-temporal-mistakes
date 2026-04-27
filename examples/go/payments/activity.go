// Package payments contains the final-state code examples for the "Activities"
// chapter of SLIDES.md. The ChargePayment activity demonstrates handling
// downstream service errors, mapping HTTP status codes to retry behavior, and
// passing a deterministic idempotency key to a Stripe-style payments API.
package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.temporal.io/sdk/temporal"

	"github.com/jlegrone/100-temporal-mistakes/internal/activityhelpers"
)

// HTTPDoer is the subset of *http.Client the activity needs. Accepting an
// interface keeps tests free of network calls.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ChargePaymentRequest models a charge in a Stripe-style payments API.
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

// Worker groups payments activities so they can share a configured client.
type Worker struct {
	Client   HTTPDoer
	Endpoint string // e.g., https://api.stripe.com/v1/charges
}

// ChargePayment posts a charge to the payments API. Status codes are mapped to
// retry behavior:
//
//   - 2xx: success
//   - 400: non-retryable application error (bad request will not succeed on retry)
//   - 429: retryable, but with a 1.5x backoff multiplier on the next retry delay
//   - other: retryable using the default backoff policy
//
// The Idempotency-Key header is derived from the workflow + activity identity
// so retries are coalesced by the upstream service.
func (w *Worker) ChargePayment(ctx context.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("marshal request: %v", err), "MarshalRequest", err,
		)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, w.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("build request: %v", err), "BuildRequest", err,
		)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotency-Key", activityhelpers.GetIdempotencyToken(ctx))

	resp, err := w.Client.Do(httpReq)
	if err != nil {
		return nil, temporal.NewApplicationErrorWithCause(
			fmt.Sprintf("call payment service: %v", err), "Network", err,
		)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		var out ChargePaymentResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, temporal.NewApplicationErrorWithCause(
				fmt.Sprintf("decode response: %v", err), "DecodeResponse", err,
			)
		}
		return &out, nil

	case resp.StatusCode == http.StatusBadRequest:
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("payment rejected: %s", readSnippet(resp.Body)),
			"BadRequest",
			nil,
		)

	case resp.StatusCode == http.StatusTooManyRequests:
		// Multiply the server-computed retry delay by 1.5 to slow down callers
		// when the upstream is overloaded.
		nextDelay := time.Duration(float64(activityhelpers.GetNextRetryDelay(ctx)) * 1.5)
		return nil, temporal.NewApplicationErrorWithOptions(
			fmt.Sprintf("rate limited: %s", readSnippet(resp.Body)),
			"RateLimited",
			temporal.ApplicationErrorOptions{NextRetryDelay: nextDelay},
		)

	default:
		return nil, temporal.NewApplicationError(
			fmt.Sprintf("payment service error %d: %s", resp.StatusCode, readSnippet(resp.Body)),
			"ServiceError",
		)
	}
}

// readSnippet returns at most 256 bytes of an error response body for inclusion
// in error messages.
func readSnippet(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 256))
	return string(bytes.TrimSpace(b))
}
