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

	"github.com/jlegrone/100-temporal-mistakes/internal/activityhelpers"
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

func TestChargePayment_DecodeTypeMismatchIsNonRetryable(t *testing.T) {
	// Server returns a 200 OK with valid JSON whose "id" field is a number,
	// not a string. This is a schema mismatch (json.UnmarshalTypeError) and
	// must not be retried.
	srv := fakeServer(t, http.StatusOK, `{"id":12345,"status":"succeeded"}`)
	w := NewWorker(srv.URL)

	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	_, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.Equal(t, "DecodeResponse", appErr.Type())
	require.True(t, appErr.NonRetryable(), "schema mismatch must be non-retryable")
}

func TestChargePayment_DecodeSyntaxErrorIsRetryable(t *testing.T) {
	// Truncated JSON body — a transient cause (network blip, server crash
	// mid-write) is plausible, so leave the error retryable.
	srv := fakeServer(t, http.StatusOK, `{"id":"ch_123",`)
	w := NewWorker(srv.URL)

	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	_, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.Equal(t, "DecodeResponse", appErr.Type())
	require.False(t, appErr.NonRetryable(), "truncated body must remain retryable")
}

func TestChargePayment_ConnectionRefusedIsRetryable(t *testing.T) {
	// Bind a port, capture its URL, then close the listener so subsequent
	// connect attempts get ECONNREFUSED.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := srv.URL
	srv.Close()

	w := NewWorker(endpoint)
	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	_, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.Equal(t, activityhelpers.HTTPTransportErrorType, appErr.Type())
	require.False(t, appErr.NonRetryable(), "connection refused must remain retryable")

	var details activityhelpers.HTTPTransportErrorDetails
	require.NoError(t, appErr.Details(&details))
	require.NotEmpty(t, details.URL)
	require.Contains(t, []string{"transport", "timeout"}, details.Reason)
}
