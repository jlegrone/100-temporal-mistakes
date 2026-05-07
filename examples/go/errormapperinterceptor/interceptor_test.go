package errormapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
)

// ----------------------------------------------------------------------
// Unit tests
// ----------------------------------------------------------------------

func TestNewErrorMapper_NameAndDelegate(t *testing.T) {
	called := 0
	want := errors.New("rewritten")
	m := NewErrorMapper("my-mapper", func(err error) (error, bool) {
		called++
		require.EqualError(t, err, "boom")
		return want, true
	})
	require.Equal(t, "my-mapper", m.Name())

	got, ok := m.MapError(errors.New("boom"))
	require.True(t, ok)
	require.Same(t, want, got)
	require.Equal(t, 1, called)
}

func TestNewErrorMapper_NilFnDeclines(t *testing.T) {
	m := NewErrorMapper("nil-fn", nil)
	got, ok := m.MapError(errors.New("anything"))
	require.False(t, ok)
	require.Nil(t, got)
}

func TestDefaultMappers(t *testing.T) {
	mappers := DefaultMappers()
	byName := make(map[string]ErrorMapper, len(mappers))
	for _, m := range mappers {
		byName[m.Name()] = m
	}

	cases := map[string]struct {
		mapperName       string
		input            error
		wantType         string
		wantNonRetryable bool
		assertDetails    func(t *testing.T, appErr *temporal.ApplicationError)
	}{
		"encoding/json UnmarshalTypeError": {
			mapperName: "encoding/json",
			input: func() error {
				var target struct {
					ID string `json:"id"`
				}
				return json.Unmarshal([]byte(`{"id":42}`), &target)
			}(),
			wantType:         "json.UnmarshalTypeError",
			wantNonRetryable: true,
		},
		"encoding/json SyntaxError": {
			mapperName: "encoding/json",
			input: func() error {
				var target struct {
					ID string `json:"id"`
				}
				return json.Unmarshal([]byte(`{"id":`), &target)
			}(),
			wantType:         "json.SyntaxError",
			wantNonRetryable: true,
		},
		"net DNSError NXDOMAIN": {
			mapperName: "net",
			input: &net.DNSError{
				Err:        "no such host",
				Name:       "missing.example",
				IsNotFound: true,
			},
			wantType:         "net.DNSError",
			wantNonRetryable: true,
			assertDetails: func(t *testing.T, appErr *temporal.ApplicationError) {
				var d dnsNotFoundDetails
				require.NoError(t, appErr.Details(&d))
				assert.Equal(t, "missing.example", d.Name)
			},
		},
		"net DNSError transient": {
			mapperName: "net",
			input: &net.DNSError{
				Err:  "server misbehaving",
				Name: "flaky.example",
			},
			wantType: "net.DNSError",
		},
		"net/url": {
			mapperName: "net/url",
			input: &url.Error{
				Op:  "Post",
				URL: "https://api.example.com/v1/charges",
				Err: errors.New("connection reset"),
			},
			wantType: "url.Error",
			assertDetails: func(t *testing.T, appErr *temporal.ApplicationError) {
				var d transportDetails
				require.NoError(t, appErr.Details(&d))
				assert.Equal(t, "Post", d.Op)
				assert.Equal(t, "https://api.example.com/v1/charges", d.URL)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m, found := byName[tc.mapperName]
			require.True(t, found, "mapper %q missing from DefaultMappers", tc.mapperName)
			mapped, ok := m.MapError(tc.input)
			require.True(t, ok, "mapper %q declined input", tc.mapperName)

			var appErr *temporal.ApplicationError
			require.True(t, errors.As(mapped, &appErr), "expected ApplicationError, got %T: %v", mapped, mapped)
			assert.Equal(t, tc.wantType, appErr.Type())
			assert.Equal(t, tc.wantNonRetryable, appErr.NonRetryable())
			if tc.assertDetails != nil {
				tc.assertDetails(t, appErr)
			}
		})
	}
}

// dynamicErrnoError is a fixture demonstrating a single mapper that derives
// the resulting Type, NonRetryable, and Details from the matched error
// itself (the gRPC-status-code parallel without taking a grpc dependency).
type dynamicErrnoError struct {
	code int
}

func (e *dynamicErrnoError) Error() string { return fmt.Sprintf("errno=%d", e.code) }

type dynamicErrnoDetails struct {
	Code int `json:"code"`
}

func TestDynamicMapper_PerErrorVerdict(t *testing.T) {
	m := NewErrorMapper("dynamic.errno", func(err error) (error, bool) {
		var e *dynamicErrnoError
		if !errors.As(err, &e) {
			return nil, false
		}
		nonRetryable := e.code >= 400 && e.code < 500
		return temporal.NewApplicationErrorWithOptions(err.Error(), fmt.Sprintf("Errno%d", e.code), temporal.ApplicationErrorOptions{
			NonRetryable: nonRetryable,
			Cause:        err,
			Details:      []interface{}{dynamicErrnoDetails{Code: e.code}},
		}), true
	})

	cases := map[string]struct {
		code             int
		wantType         string
		wantNonRetryable bool
	}{
		"client error 404": {code: 404, wantType: "Errno404", wantNonRetryable: true},
		"server error 503": {code: 503, wantType: "Errno503", wantNonRetryable: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mapped, ok := m.MapError(&dynamicErrnoError{code: tc.code})
			require.True(t, ok)
			var appErr *temporal.ApplicationError
			require.True(t, errors.As(mapped, &appErr))
			assert.Equal(t, tc.wantType, appErr.Type())
			assert.Equal(t, tc.wantNonRetryable, appErr.NonRetryable())
			var d dynamicErrnoDetails
			require.NoError(t, appErr.Details(&d))
			assert.Equal(t, tc.code, d.Code)
		})
	}
}

func TestIsNativeTemporalError(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"plain stdlib error": {err: errors.New("boom"), want: false},
		"json type error":    {err: &json.UnmarshalTypeError{}, want: false},
		"ApplicationError":   {err: temporal.NewApplicationError("boom", "Boom"), want: true},
		"NonRetryable ApplicationError": {
			err: temporal.NewNonRetryableApplicationError("boom", "Boom", nil), want: true,
		},
		"CanceledError":  {err: temporal.NewCanceledError(), want: true},
		"wrapped AppErr": {err: fmt.Errorf("wrap: %w", temporal.NewApplicationError("boom", "Boom")), want: true},
		"context.Canceled is not native temporal": {err: context.Canceled, want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, isNativeTemporalError(tc.err))
		})
	}
}

// ----------------------------------------------------------------------
// Integration tests via TestActivityEnvironment
// ----------------------------------------------------------------------

func newActivityEnv(t *testing.T, mappers ...ErrorMapper) *testsuite.TestActivityEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{New(Options{Mappers: mappers})},
	})
	return env
}

func TestIntegration_ConvertsPlainError(t *testing.T) {
	act := func(context.Context) error {
		var target struct {
			ID string `json:"id"`
		}
		return json.Unmarshal([]byte(`{"id":42}`), &target)
	}
	env := newActivityEnv(t, DefaultMappers()...)
	env.RegisterActivity(act)

	_, err := env.ExecuteActivity(act)
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr), "expected ApplicationError, got %T: %v", err, err)
	assert.Equal(t, "json.UnmarshalTypeError", appErr.Type())
	assert.True(t, appErr.NonRetryable())
}

func TestIntegration_NativePassThrough(t *testing.T) {
	// A mapper that would otherwise rewrite *any* error — confirms the
	// native pass-through short-circuits before the chain runs.
	greedy := NewErrorMapper("greedy", func(err error) (error, bool) {
		return temporal.NewApplicationError("rewritten", "Rewritten"), true
	})

	cases := map[string]error{
		"ApplicationError": temporal.NewNonRetryableApplicationError("kept", "OriginalType", nil),
		"CanceledError":    temporal.NewCanceledError(),
	}
	for name, native := range cases {
		t.Run(name, func(t *testing.T) {
			activityErr := native // capture for closure
			act := func(context.Context) error { return activityErr }

			env := newActivityEnv(t, greedy)
			env.RegisterActivity(act)
			_, err := env.ExecuteActivity(act)
			require.Error(t, err)

			var appErr *temporal.ApplicationError
			if errors.As(err, &appErr) {
				assert.NotEqual(t, "Rewritten", appErr.Type(), "greedy mapper must not rewrite native error")
			}
			// Whatever the surfaced concrete type, it must still be a native
			// Temporal error — not the greedy mapper's "Rewritten" output.
			assert.True(t, isNativeTemporalError(err), "expected native temporal error to pass through, got %T: %v", err, err)
		})
	}
}

func TestIntegration_FirstMatchWins(t *testing.T) {
	matchTypeErr := func(name, errType string) ErrorMapper {
		return NewErrorMapper(name, func(err error) (error, bool) {
			var t *json.UnmarshalTypeError
			if !errors.As(err, &t) {
				return nil, false
			}
			return temporal.NewApplicationError(err.Error(), errType), true
		})
	}
	first := matchTypeErr("first", "First")
	second := matchTypeErr("second", "Second")

	act := func(context.Context) error {
		var target struct {
			ID string `json:"id"`
		}
		return json.Unmarshal([]byte(`{"id":42}`), &target)
	}
	env := newActivityEnv(t, first, second)
	env.RegisterActivity(act)

	_, err := env.ExecuteActivity(act)
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, "First", appErr.Type())
}

func TestIntegration_UnmatchedPassthrough(t *testing.T) {
	sentinel := errors.New("upstream boom")
	act := func(context.Context) error { return sentinel }

	nonMatching := NewErrorMapper("never", func(err error) (error, bool) { return nil, false })

	env := newActivityEnv(t, nonMatching)
	env.RegisterActivity(act)

	_, err := env.ExecuteActivity(act)
	require.Error(t, err)
	// The SDK round-trips the error through its failure converter so
	// errors.Is can't reach the original sentinel by identity; the
	// observable contract is that the original message survives untouched
	// (no mapper rewrote the type).
	require.Contains(t, err.Error(), sentinel.Error())
}
