package doing_work_outside_of_the_workflow

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// Data represents input data for the workflow.
type Data struct{}

// Record represents a database record.
type Record struct {
	ID string
}

type database struct{}

func (database) Insert(_ context.Context, _ Data) (Record, error) {
	return Record{}, nil
}

// InsertRecord is an activity that inserts a record into the database.
func InsertRecord(_ context.Context, _ Data) (Record, error) {
	return Record{}, nil
}

// @@@SNIPSTART doing-work-outside-bad
// Dangerous: work done outside the workflow
func dangerousExample(ctx context.Context, temporalClient client.Client, db database, data Data, options client.StartWorkflowOptions) error {
	record, err := db.Insert(ctx, data)
	if err != nil {
		return err
	}
	// If the process crashes HERE, the record exists but no workflow was started
	_, err = temporalClient.ExecuteWorkflow(ctx, options, MyWorkflow, record.ID)
	return err
}

// @@@SNIPEND

// @@@SNIPSTART doing-work-outside-good
// Better: start the workflow first, let it do the work durably
func betterExample(ctx context.Context, temporalClient client.Client, data Data, options client.StartWorkflowOptions) error {
	_, err := temporalClient.ExecuteWorkflow(ctx, options, MyWorkflow, data)
	return err
}

func MyWorkflow(ctx workflow.Context, data Data) error {
	var record Record
	err := workflow.ExecuteActivity(ctx, InsertRecord, data).Get(ctx, &record)
	// If the worker crashes after insert, replay skips the completed activity
	_ = record
	return err
}

// @@@SNIPEND
