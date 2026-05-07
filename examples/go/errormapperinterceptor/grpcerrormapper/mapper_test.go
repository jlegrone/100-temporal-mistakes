package grpcerrormapper

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestMapper_Name(t *testing.T) {
	assert.Equal(t, MapperName, Mapper().Name())
	assert.Equal(t, "google.golang.org/grpc", MapperName)
}

func TestMapper_CodeMapping(t *testing.T) {
	cases := map[string]struct {
		code             codes.Code
		wantNonRetryable bool
	}{
		// Retryable
		"Canceled":          {code: codes.Canceled, wantNonRetryable: false},
		"DeadlineExceeded":  {code: codes.DeadlineExceeded, wantNonRetryable: false},
		"Unavailable":       {code: codes.Unavailable, wantNonRetryable: false},
		"ResourceExhausted": {code: codes.ResourceExhausted, wantNonRetryable: false},
		"Aborted":           {code: codes.Aborted, wantNonRetryable: false},
		"Internal":          {code: codes.Internal, wantNonRetryable: false},
		"Unknown":           {code: codes.Unknown, wantNonRetryable: false},
		"DataLoss":          {code: codes.DataLoss, wantNonRetryable: false},
		// Non-retryable
		"InvalidArgument":    {code: codes.InvalidArgument, wantNonRetryable: true},
		"NotFound":           {code: codes.NotFound, wantNonRetryable: true},
		"AlreadyExists":      {code: codes.AlreadyExists, wantNonRetryable: true},
		"FailedPrecondition": {code: codes.FailedPrecondition, wantNonRetryable: true},
		"OutOfRange":         {code: codes.OutOfRange, wantNonRetryable: true},
		"Unauthenticated":    {code: codes.Unauthenticated, wantNonRetryable: true},
		"PermissionDenied":   {code: codes.PermissionDenied, wantNonRetryable: true},
		"Unimplemented":      {code: codes.Unimplemented, wantNonRetryable: true},
	}

	m := Mapper()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			input := status.Error(tc.code, "boom")
			mapped, ok := m.MapError(input)
			require.True(t, ok, "mapper declined %s", tc.code)

			var appErr *temporal.ApplicationError
			require.True(t, errors.As(mapped, &appErr), "expected ApplicationError, got %T: %v", mapped, mapped)
			assert.Equal(t, "grpc."+tc.code.String(), appErr.Type())
			assert.Equal(t, tc.wantNonRetryable, appErr.NonRetryable())

			var d *spb.Status
			require.NoError(t, appErr.Details(&d))
			assert.Equal(t, int32(tc.code), d.GetCode())
			assert.Equal(t, "boom", d.GetMessage())
		})
	}
}

func TestMapper_DeclinesNonStatusError(t *testing.T) {
	mapped, ok := Mapper().MapError(errors.New("plain"))
	assert.False(t, ok)
	assert.Nil(t, mapped)
}

func TestMapper_DeclinesNilError(t *testing.T) {
	// status.FromError(nil) returns (nil, true) with codes.OK; the mapper
	// must decline rather than fabricate an ApplicationError out of OK.
	mapped, ok := Mapper().MapError(nil)
	assert.False(t, ok)
	assert.Nil(t, mapped)
}

func TestMapper_PreservesCause(t *testing.T) {
	cause := status.Error(codes.Unavailable, "upstream down")
	mapped, ok := Mapper().MapError(cause)
	require.True(t, ok)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(mapped, &appErr))
	require.NotNil(t, appErr.Unwrap(), "Cause must be preserved")
	assert.ErrorIs(t, mapped, cause)
}

func TestMapper_RetryInfoSetsNextRetryDelay(t *testing.T) {
	st, err := status.New(codes.ResourceExhausted, "slow down").WithDetails(&errdetails.RetryInfo{
		RetryDelay: durationpb.New(7 * time.Second),
	})
	require.NoError(t, err)

	mapped, ok := Mapper().MapError(st.Err())
	require.True(t, ok)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(mapped, &appErr))
	assert.Equal(t, 7*time.Second, appErr.NextRetryDelay())
}

func TestMapper_RetryInfoZeroOrMissingLeavesDelayUnset(t *testing.T) {
	cases := map[string]*errdetails.RetryInfo{
		"missing RetryDelay":  {},
		"zero RetryDelay":     {RetryDelay: durationpb.New(0)},
		"negative RetryDelay": {RetryDelay: durationpb.New(-5 * time.Second)},
	}
	for name, ri := range cases {
		t.Run(name, func(t *testing.T) {
			st, err := status.New(codes.Unavailable, "transient").WithDetails(ri)
			require.NoError(t, err)

			mapped, ok := Mapper().MapError(st.Err())
			require.True(t, ok)
			var appErr *temporal.ApplicationError
			require.True(t, errors.As(mapped, &appErr))
			assert.Zero(t, appErr.NextRetryDelay())
		})
	}
}

func TestMapper_RetryInfoWithoutDetails(t *testing.T) {
	mapped, ok := Mapper().MapError(status.Error(codes.Unavailable, "transient"))
	require.True(t, ok)
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(mapped, &appErr))
	assert.Zero(t, appErr.NextRetryDelay())
}
