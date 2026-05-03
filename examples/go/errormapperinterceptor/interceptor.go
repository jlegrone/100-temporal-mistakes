// Package errormapper is a worker interceptor that converts plain errors
// returned from activities into structured temporal.ApplicationErrors via a
// configurable, ordered list of named [ErrorMapper]s.
//
// A single mapper can claim a whole family of errors and derive the
// resulting Type, NonRetryable, and Details payload from the matched value
// (e.g. a gRPC mapper that produces a different ApplicationError per
// status.Code). Mappers are evaluated in order; the first ok=true wins.
//
// Native Temporal SDK error types — *temporal.ApplicationError,
// *temporal.CanceledError, *temporal.TimeoutError, *temporal.TerminatedError
// — bypass the mapper chain entirely and are returned to the SDK unchanged.
// They have already been classified by the activity author or the SDK
// itself; rewriting them would mask retryability, cancellation, and timeout
// semantics. This is an invariant, not a configurable option.
package errormapper

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"

	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
)

// ErrorMapper inspects an error returned from an activity and either
// rewrites it (ok=true) or declines (ok=false), letting subsequent mappers
// in the chain try. A single ErrorMapper may handle a whole family of errors
// — for instance, one mapper that produces a per-status-code
// ApplicationError for every grpc.Status error.
//
// Implementations are responsible for preserving the original error as the
// cause via [temporal.ApplicationErrorOptions.Cause] when constructing the
// rewritten error.
type ErrorMapper interface {
	// Name uniquely identifies the mapper for diagnostics, telemetry, and
	// logging. Stable across runs.
	Name() string
	// MapError returns the rewritten error and ok=true when this mapper
	// claims err, or (nil, false) to decline.
	MapError(err error) (mapped error, ok bool)
}

// NewErrorMapper returns an ErrorMapper that delegates to fn for matching
// and rewriting. name is exposed via [ErrorMapper.Name].
func NewErrorMapper(name string, fn func(err error) (mapped error, ok bool)) ErrorMapper {
	return mapperFunc{name: name, fn: fn}
}

type mapperFunc struct {
	name string
	fn   func(err error) (error, bool)
}

func (m mapperFunc) Name() string { return m.name }

func (m mapperFunc) MapError(err error) (error, bool) {
	if m.fn == nil {
		return nil, false
	}
	return m.fn(err)
}

// Options configures the interceptor.
type Options struct {
	// Mappers is evaluated in order; the first ok=true wins. Errors that no
	// mapper claims pass through to the SDK untouched. Native Temporal SDK
	// error types bypass the chain entirely (see package doc).
	Mappers []ErrorMapper
}

// New returns a [interceptor.WorkerInterceptor] that rewrites activity
// errors per opts.
//
// TODO(jlegrone): convert this to a WorkerPlugin once the API stabilizes
// (currently in go.temporal.io/sdk/internal#WorkerPlugin).
func New(opts Options) interceptor.WorkerInterceptor {
	return &errorMapperInterceptor{opts: opts}
}

type errorMapperInterceptor struct {
	interceptor.WorkerInterceptorBase
	opts Options
}

func (i *errorMapperInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &activityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		opts:                           i.opts,
	}
}

type activityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	opts Options
}

func (a *activityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (interface{}, error) {
	out, err := a.Next.ExecuteActivity(ctx, in)
	if err == nil || isNativeTemporalError(err) {
		return out, err
	}
	for _, m := range a.opts.Mappers {
		if mapped, ok := m.MapError(err); ok {
			return out, mapped
		}
	}
	return out, err
}

// isNativeTemporalError reports whether err already is (or wraps) a native
// Temporal SDK error type that the interceptor must leave untouched.
func isNativeTemporalError(err error) bool {
	var (
		appErr *temporal.ApplicationError
		canErr *temporal.CanceledError
		toErr  *temporal.TimeoutError
		tmErr  *temporal.TerminatedError
	)
	return errors.As(err, &appErr) ||
		errors.As(err, &canErr) ||
		errors.As(err, &toErr) ||
		errors.As(err, &tmErr)
}

// dnsNotFoundDetails is the details payload attached by the "net.dns"
// default mapper. The exported field tags are the over-the-wire contract;
// consumers decode into a struct of their own with matching json tags.
type dnsNotFoundDetails struct {
	Name string `json:"name"`
}

// transportDetails is the details payload attached by the "net.url" default
// mapper. The exported field tags are the over-the-wire contract; consumers
// decode into a struct of their own with matching json tags.
type transportDetails struct {
	Op  string `json:"op"`
	URL string `json:"url"`
}

// checkError reports whether err errors.As-matches T and returns the matched
// value (the zero value when ok is false). It exists so default mappers can
// dispatch over their package's error types with a series of typed checks
// rather than re-declaring local target variables for each one.
func checkError[T error](err error) (T, bool) {
	var typedErr T
	if errors.As(err, &typedErr) {
		return typedErr, true
	}
	return typedErr, false
}

// DefaultMappers returns ErrorMappers for common stdlib error types this
// codebase already uses. There is one mapper per standard library package,
// named after that package's import path ("encoding/json", "net",
// "net/url").
//
// Order is important: more specific packages come first so that, for
// example, a *url.Error that wraps a *net.DNSError is claimed by "net"
// before the generic "net/url" backstop.
func DefaultMappers() []ErrorMapper {
	return []ErrorMapper{
		NewErrorMapper("encoding/json", func(err error) (error, bool) {
			if err, ok := checkError[*json.UnmarshalTypeError](err); ok {
				return temporal.NewApplicationErrorWithOptions(err.Error(), "json.UnmarshalTypeError", temporal.ApplicationErrorOptions{
					// Schema disagreement between sender and receiver;
					// neither side will change shape on retry.
					NonRetryable: true,
					Cause:        err,
				}), true
			}
			if err, ok := checkError[*json.SyntaxError](err); ok {
				return temporal.NewApplicationErrorWithOptions(err.Error(), "json.SyntaxError", temporal.ApplicationErrorOptions{
					// Malformed JSON in a fully-buffered payload — the
					// bytes are deterministic, so retrying parses the
					// same garbage again. (Streaming I/O truncation
					// surfaces as io.ErrUnexpectedEOF, not SyntaxError.)
					NonRetryable: true,
					Cause:        err,
				}), true
			}
			return nil, false
		}),
		NewErrorMapper("net", func(err error) (error, bool) {
			if dnsErr, ok := checkError[*net.DNSError](err); ok {
				return temporal.NewApplicationErrorWithOptions(err.Error(), "net.DNSError", temporal.ApplicationErrorOptions{
					// IsNotFound means NXDOMAIN: the configured host
					// genuinely doesn't exist. Other DNS failures
					// (SERVFAIL, timeouts) can be transient, so leave
					// those retryable.
					NonRetryable: dnsErr.IsNotFound,
					Cause:        err,
					Details:      []interface{}{dnsNotFoundDetails{Name: dnsErr.Name}},
				}), true
			}
			return nil, false
		}),
		NewErrorMapper("net/url", func(err error) (error, bool) {
			if urlErr, ok := checkError[*url.Error](err); ok {
				return temporal.NewApplicationErrorWithOptions(err.Error(), "url.Error", temporal.ApplicationErrorOptions{
					Cause:   err,
					Details: []interface{}{transportDetails{Op: urlErr.Op, URL: urlErr.URL}},
				}), true
			}
			return nil, false
		}),
	}
}
