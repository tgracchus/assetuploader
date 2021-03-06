package job_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tgracchus/assetuploader/pkg/job"
	"github.com/tgracchus/assetuploader/pkg/util"
)

var jobTimeout = 500 * time.Millisecond
var waitTime = 100 * time.Millisecond

var testJobFunction = func(ctx context.Context) error {
	return nil
}

func TestAddAndGetJob(t *testing.T) {
	upsert, query := job.NewMemoryStore(job.MillisKeys)
	executionDate := time.Now()
	ctx := context.Background()
	expectedJob := job.NewFixedDateJob(uuid.New().String(), testJobFunction, executionDate)
	err := job.UpSert(ctx, upsert, *expectedJob)
	if err != nil {
		t.Fatal(err)
	}
	var jobs []job.Job
	err = util.WaitUntilWithContext(ctx, func(ctx context.Context) error {
		jobs, err = job.GetBefore(ctx, query, executionDate, newStoreTestCriteria(expectedJob.Status))
		if err != nil {
			return err
		}
		if len(jobs) != 1 {
			return errors.New("Expected at least one job")
		}
		return nil
	}, waitTime, jobTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if jobs[0].ID != expectedJob.ID {
		t.Fatal("Expected job and actual job do not match")
	}
}

func TestGetBeforeCancelled(t *testing.T) {
	_, query := job.NewMemoryStore(job.MillisKeys)
	executionDate := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := job.GetBefore(ctx, query, executionDate, newStoreTestCriteria(job.ErrorStatus))
	if err == nil {
		t.Fatal(err)
	}
	if err.Error() != "context canceled" {
		t.Fatalf("We expect the context to be cancelled")
	}
}

func TestUpsetCancelled(t *testing.T) {
	upsert, _ := job.NewMemoryStore(job.MillisKeys)
	executionDate := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	expectedJob := job.NewFixedDateJob(uuid.New().String(), testJobFunction, executionDate)
	err := job.UpSert(ctx, upsert, *expectedJob)
	if err == nil {
		t.Fatal(err)
	}
	if err.Error() != "context canceled" {
		t.Fatalf("We expect the context to be cancelled")
	}
}

func TestUpdateJobStatus(t *testing.T) {
	upsert, query := job.NewMemoryStore(job.MillisKeys)
	executionDate := time.Now()
	ctx := context.Background()
	newJob := job.NewFixedDateJob(uuid.New().String(), testJobFunction, executionDate)
	err := job.UpSert(ctx, upsert, *newJob)
	if err != nil {
		t.Fatal(err)
	}
	var foundJobs []job.Job
	err = util.WaitUntilWithContext(ctx, func(ctx context.Context) error {
		foundJobs, err = job.GetBefore(ctx, query, executionDate, newStoreTestCriteria(newJob.Status))
		if err != nil {
			return err
		}
		if len(foundJobs) != 1 {
			return errors.New("Expected at least one job")
		}
		return nil
	}, waitTime, jobTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if len(foundJobs) != 1 {
		t.Fatal("Expected at least one job")
	}

	updatedJob := newJob.Executing()
	err = job.UpSert(ctx, upsert, updatedJob)
	if err != nil {
		t.Fatal(err)
	}
	var updatedFoundJobs []job.Job
	err = util.WaitUntilWithContext(ctx, func(ctx context.Context) error {
		updatedFoundJobs, err = job.GetBefore(ctx, query, executionDate, newStoreTestCriteria(updatedJob.Status))
		if err != nil {
			return err
		}
		if len(updatedFoundJobs) != 1 {
			return errors.New("Expected at least one job")
		}
		return nil
	}, waitTime, jobTimeout)
	if err != nil {
		t.Fatal(err)
	}
	foundJob := foundJobs[0]
	if foundJob.ID != newJob.ID {
		t.Fatal("Expected job and actual job do not match")
	}
	if !newJob.IsNew() {
		t.Fatal("Expected job should be in scheduled state")
	}
	if !foundJob.IsNew() {
		t.Fatal("Expected job should be in scheduled state")
	}
	updatedFoundJob := updatedFoundJobs[0]
	if updatedFoundJob.ID != newJob.ID {
		t.Fatal("Expected job and actual job do not match")
	}
	if updatedFoundJob.ID != updatedJob.ID {
		t.Fatal("Expected job and actual job do not match")
	}
	if !newJob.IsNew() {
		t.Fatal("Expected job should be in scheduled state")
	}
	if !updatedJob.IsExecuting() {
		t.Fatal("Expected job should be in scheduled state")
	}
	if !updatedFoundJob.IsExecuting() {
		t.Fatal("Expected job should be in scheduled state")
	}
}

func TestAddJobPastInTime(t *testing.T) {
	upsert, query := job.NewMemoryStore(job.MillisKeys)
	now := time.Now()
	pastExecutionDate := now.Add(-1 * time.Hour)
	ctx := context.Background()
	pastJob := job.NewFixedDateJob(uuid.New().String(), testJobFunction, pastExecutionDate)
	err := job.UpSert(ctx, upsert, *pastJob)
	if err != nil {
		t.Fatal(err)
	}
	newJob := job.NewFixedDateJob(uuid.New().String(), testJobFunction, now)
	err = job.UpSert(ctx, upsert, *newJob)
	if err != nil {
		t.Fatal(err)
	}

	// Since JobStore follows PRAM consistency model,
	// we need to wait for the add channel to be drained, so we can observe the two jobs
	var foundJobs []job.Job
	err = util.WaitUntilWithContext(ctx, func(ctx context.Context) error {
		foundJobs, err = job.GetBefore(ctx, query, now, newStoreTestCriteria(newJob.Status))
		if err != nil {
			return err
		}
		if len(foundJobs) != 2 {
			return errors.New("Expected at least two jobs")
		}
		return nil
	}, waitTime, jobTimeout)
	if err != nil {
		t.Fatal(err)
	}
	foundJob := foundJobs[0]
	if newJob.ID != foundJob.ID {
		t.Fatal("Expected job and actual job do not match")
	}
}

func newStoreTestCriteria(status job.Status) func(job job.Job) bool {
	return func(job job.Job) bool {
		return job.Status == status
	}
}
