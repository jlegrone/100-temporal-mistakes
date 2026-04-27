package payments

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type fakeDoer struct {
	resp *http.Response
	err  error
	got  *http.Request
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.got = req
	return f.resp, f.err
}

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func newActivityEnv() *testsuite.TestActivityEnvironment {
	suite := &testsuite.WorkflowTestSuite{}
	return suite.NewTestActivityEnvironment()
}

func TestChargePayment_Success(t *testing.T) {
	doer := &fakeDoer{resp: newResponse(http.StatusOK, `{"id":"ch_123","status":"succeeded"}`)}
	w := &Worker{Client: doer, Endpoint: "https://api.example/charges"}

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

	require.NotNil(t, doer.got)
	require.NotEmpty(t, doer.got.Header.Get("Idempotency-Key"))
	require.Equal(t, "application/json", doer.got.Header.Get("Content-Type"))
}

func TestChargePayment_BadRequestIsNonRetryable(t *testing.T) {
	doer := &fakeDoer{resp: newResponse(http.StatusBadRequest, "missing currency")}
	w := &Worker{Client: doer, Endpoint: "https://api.example/charges"}

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
	doer := &fakeDoer{resp: newResponse(http.StatusTooManyRequests, "slow down")}
	w := &Worker{Client: doer, Endpoint: "https://api.example/charges"}

	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	_, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	require.False(t, appErr.NonRetryable())
	require.Equal(t, "RateLimited", appErr.Type())
}

func TestChargePayment_GenericServerErrorIsRetryable(t *testing.T) {
	doer := &fakeDoer{resp: newResponse(http.StatusInternalServerError, "boom")}
	w := &Worker{Client: doer, Endpoint: "https://api.example/charges"}

	env := newActivityEnv()
	env.RegisterActivity(w.ChargePayment)
	_, err := env.ExecuteActivity(w.ChargePayment, ChargePaymentRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		require.False(t, appErr.NonRetryable())
	}
}
