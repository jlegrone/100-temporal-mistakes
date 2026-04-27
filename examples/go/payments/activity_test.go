package payments

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func newActivityEnv() *testsuite.TestActivityEnvironment {
	suite := &testsuite.WorkflowTestSuite{}
	return suite.NewTestActivityEnvironment()
}

// fakeServer starts an httptest.Server that responds to every request with
// the given status code and body.
func fakeServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestChargePayment_Success(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ch_123","status":"succeeded"}`)
	}))
	t.Cleanup(srv.Close)

	w := NewWorker(srv.URL)
	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	val, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{
		CustomerID:  "cus_1",
		AmountCents: 500,
		Currency:    "usd",
	})
	require.NoError(t, err)

	var resp ChargePaymentResponse
	require.NoError(t, val.Get(&resp))
	require.Equal(t, "ch_123", resp.ChargeID)
	require.Equal(t, "succeeded", resp.Status)

	require.NotNil(t, got)
	require.NotEmpty(t, got.Header.Get("Idempotency-Key"))
	require.Equal(t, "application/json", got.Header.Get("Content-Type"))
}

func TestChargePayment_BadRequestIsNonRetryable(t *testing.T) {
	srv := fakeServer(t, http.StatusBadRequest, "missing currency")
	w := NewWorker(srv.URL)

	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	_, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, "BadRequest", appErr.Type())
}

func TestChargePayment_RateLimitedReturnsApplicationError(t *testing.T) {
	srv := fakeServer(t, http.StatusTooManyRequests, "slow down")
	w := NewWorker(srv.URL)

	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	_, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.False(t, appErr.NonRetryable())
	require.Equal(t, "TooManyRequests", appErr.Type())
}

func TestChargePayment_GenericServerErrorIsRetryable(t *testing.T) {
	srv := fakeServer(t, http.StatusInternalServerError, "boom")
	w := NewWorker(srv.URL)

	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	_, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		require.False(t, appErr.NonRetryable())
	}
}
