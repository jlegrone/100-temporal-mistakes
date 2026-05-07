// Package activitypolicyinterceptor exports the SDK-agnostic conformance
// test JSON for the Activity Policy Interceptor specification.
//
// The spec itself is the README.md alongside this file. Implementations in
// any host SDK should parse [ConformanceJSON] to drive their conformance
// test runner. The Go reference implementation lives at
// examples/go/activitypolicyinterceptor.
package activitypolicyinterceptor

import _ "embed"

// ConformanceJSON is the raw bytes of conformance_tests.json. The contents
// follow the schema documented at the top of that file.
//
//go:embed conformance_tests.json
var ConformanceJSON []byte
