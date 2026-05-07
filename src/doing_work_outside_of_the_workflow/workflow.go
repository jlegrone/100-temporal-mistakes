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

// Database is a stub for a database client.
type Database interface {
	Insert(ctx context.Context, data Data) (Record, error)
}

// InsertRecord is an activity that inserts a record into the database.
func InsertRecord(ctx context.Context, d Data) (Record, error) {
	return Record{ID: "foo"}, nil
}

// MyService holds the dependencies needed to start the workflow.
type MyService struct {
	Temporal client.Client
	DB       Database
}

// @@@SNIPSTART doing-work-outside-bad

// HandleRequest performs a database insert before starting the workflow.
// If the process crashes between the insert and the workflow start, the
// record exists but no workflow is running to process it.
func (s *MyService) HandleRequest(ctx context.Context, data Data) error {
	record, err := s.DB.Insert(ctx, data)
	if err != nil {
		return err
	}
	// If the process crashes HERE, the record exists but no workflow was started.
	if _, err := s.Temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: "my-task-queue",
	}, MyWorkflowV1, record.ID); err != nil {
		return err
	}
	return nil
}

// MyWorkflowV1 receives the already-inserted record ID.
func MyWorkflowV1(ctx workflow.Context, id string) error {
	// ... process the record

	return nil
}

// @@@SNIPEND

// @@@SNIPSTART doing-work-outside-good

// HandleRequestV2 starts the workflow first and lets it perform the
// database insert as an activity, so both operations are durable.
func (s *MyService) HandleRequestV2(ctx context.Context, data Data) error {
	if _, err := s.Temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: "my-task-queue",
	}, MyWorkflowV2, data); err != nil {
		return err
	}
	return nil
}

func MyWorkflowV2(ctx workflow.Context, data Data) error {
	var record Record
	if err := workflow.ExecuteActivity(ctx, InsertRecord, data).Get(ctx, &record); err != nil {
		return err
	}

	// ... continue processing with record.ID

	return nil
}

// @@@SNIPEND
