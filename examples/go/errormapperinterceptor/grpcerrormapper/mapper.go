// Package grpcerrormapper provides an [errormapper.ErrorMapper] that
// rewrites gRPC status errors into structured [temporal.ApplicationError]s.
//
// Codes are mapped to retryability per the canonical guidance behind
// [gRFC A6]: client errors (e.g. InvalidArgument, NotFound) are
// non-retryable; server errors and transient codes (Unavailable,
// ResourceExhausted, Aborted, Internal, Unknown, DataLoss, plus Canceled
// and DeadlineExceeded) stay retryable so the activity's RetryPolicy
// applies.
//
// When the server attaches a [google.rpc.RetryInfo] to the status details,
// its retry_delay is forwarded as
// [temporal.ApplicationErrorOptions.NextRetryDelay] so callers honor the
// upstream's hint.
//
// A6's grpc-retry-pushback-ms trailer is intentionally not consulted: it
// is delivered as trailing metadata that gRPC's built-in retry machinery
// consumes before the application sees the error, so it is not available
// on the [google.golang.org/grpc/status.Status] returned to callers.
// RetryInfo (in status details) is the application-visible parallel.
//
// [gRFC A6]: https://github.com/grpc/proposal/blob/master/A6-client-retries.md
package grpcerrormapper

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errormapper "github.com/jlegrone/100-temporal-mistakes/examples/go/errormapperinterceptor"
)

// MapperName is the [errormapper.ErrorMapper.Name] returned by [Mapper].
// It follows the import-path convention used by the parent package's
// DefaultMappers (e.g. "encoding/json", "net/url").
const MapperName = "google.golang.org/grpc"

// nonRetryableCodes lists gRPC codes whose semantics indicate the
// request will not succeed on retry without changing the request itself.
// All other codes (including Canceled and DeadlineExceeded) stay
// retryable so the activity's RetryPolicy applies.
var nonRetryableCodes = map[codes.Code]struct{}{
	codes.InvalidArgument:    {},
	codes.NotFound:           {},
	codes.AlreadyExists:      {},
	codes.FailedPrecondition: {},
	codes.OutOfRange:         {},
	codes.Unauthenticated:    {},
	codes.PermissionDenied:   {},
	codes.Unimplemented:      {},
}

// Mapper returns an [errormapper.ErrorMapper] that claims gRPC status
// errors. Errors that are not gRPC statuses (or that carry codes.OK)
// are declined.
func Mapper() errormapper.ErrorMapper {
	return errormapper.NewErrorMapper(MapperName, mapError)
}

func mapError(err error) (error, bool) {
	st, ok := status.FromError(err)
	if !ok || st.Code() == codes.OK {
		return nil, false
	}

	code := st.Code()
	_, nonRetryable := nonRetryableCodes[code]

	opts := temporal.ApplicationErrorOptions{
		NonRetryable: nonRetryable,
		Cause:        err,
		// Attach the canonical google.rpc.Status proto rather than a
		// custom struct: it already carries code, message, and any
		// server-supplied details, and consumers can deserialize it
		// with the standard proto tooling.
		Details: []interface{}{st.Proto()},
	}
	if delay := retryDelayFromStatus(st); delay > 0 {
		opts.NextRetryDelay = delay
	}

	return temporal.NewApplicationErrorWithOptions(
		st.Message(),
		"grpc."+code.String(),
		opts,
	), true
}

// retryDelayFromStatus returns the first positive RetryInfo.retry_delay
// from the status details, or zero if none is present.
func retryDelayFromStatus(st *status.Status) time.Duration {
	for _, d := range st.Details() {
		ri, ok := d.(*errdetails.RetryInfo)
		if !ok || ri.GetRetryDelay() == nil {
			continue
		}
		if got := ri.GetRetryDelay().AsDuration(); got > 0 {
			return got
		}
	}
	return 0
}
