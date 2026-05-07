package workflowhelpers

import (
	"github.com/oklog/ulid/v2"
	"go.temporal.io/sdk/workflow"
)

// NewULID returns a [ulid.ULID] with the current time in Unix milliseconds and
// monotonically increasing entropy for the same millisecond.
//
// ULIDs have the following properties which may make them preferable over UUID/GUID:
//   - 128-bit compatibility with UUID
//   - 1.21e+24 unique ULIDs per millisecond
//   - Lexicographically sortable!
//   - Canonically encoded as a 26 character string, as opposed to the 36 character UUID
//   - Use Crockford's base32 for better efficiency and readability (5 bits per character)
//   - Case insensitive
//   - No special characters (URL safe)
//   - Monotonic sort order (correctly detects and handles the same millisecond)
//
// If you don't care about time-based ordering of generated IDs, then there's no
// reason to use ULIDs! There are many other kinds of IDs that are easier, faster,
// smaller, etc. Consider UUIDs.
//
// Reference: https://github.com/ulid/spec
func NewULID(ctx workflow.Context) (ulid.ULID, error) {
	raw, err := SideEffect(ctx, func(workflow.Context) string {
		//workflowcheck:ignore reason: calls non-deterministic function ulid.Make
		return ulid.Make().String()
	})
	if err != nil {
		return ulid.ULID{}, err
	}
	return ulid.Parse(raw)
}
