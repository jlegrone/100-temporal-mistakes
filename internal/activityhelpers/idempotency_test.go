package activityhelpers

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

func makeInfo(workflowID, runID, activityID string) activity.Info {
	return activity.Info{
		WorkflowExecution: workflow.Execution{ID: workflowID, RunID: runID},
		ActivityID:        activityID,
	}
}

func TestGetIdempotencyToken_Deterministic(t *testing.T) {
	a := getIdempotencyToken(makeInfo("wf-1", "run-1", "act-1"))
	b := getIdempotencyToken(makeInfo("wf-1", "run-1", "act-1"))
	assert.Equal(t, a, b)
}

func TestGetIdempotencyToken_Format(t *testing.T) {
	token := getIdempotencyToken(makeInfo("wf-1", "run-1", "act-1"))
	// SHA-256 hex encoding is 64 characters.
	assert.Len(t, token, 2*sha256.Size)
	_, err := hex.DecodeString(token)
	assert.NoError(t, err)
}

func TestGetIdempotencyToken_Distinct(t *testing.T) {
	tests := map[string]activity.Info{
		"baseline":             makeInfo("wf", "run", "act"),
		"different workflow":   makeInfo("wf2", "run", "act"),
		"different run":        makeInfo("wf", "run2", "act"),
		"different activity":   makeInfo("wf", "run", "act2"),
		"empty workflow":       makeInfo("", "run", "act"),
		"empty run":            makeInfo("wf", "", "act"),
		"empty activity":       makeInfo("wf", "run", ""),
		"all empty":            makeInfo("", "", ""),
		"boundary collision a": makeInfo("foo", "run", "bar"),
		"boundary collision b": makeInfo("foob", "run", "ar"),
	}

	seen := make(map[string]string, len(tests))
	for name, info := range tests {
		token := getIdempotencyToken(info)
		if existing, ok := seen[token]; ok {
			t.Errorf("token collision: %q and %q both produced %s", existing, name, token)
		}
		seen[token] = name
	}
}

func TestGetIdempotencyToken_NonActivityContext(t *testing.T) {
	assert.NotPanics(t, func() {
		assert.Equal(t, "", GetIdempotencyToken(t.Context()))
	})
}
