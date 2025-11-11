package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueue_EnqueueAndProcess(t *testing.T) {
	processed := int32(0)

	handler := func(ctx context.Context, req interface{}) (string, error) {
		atomic.AddInt32(&processed, 1)
		return "msg-123", nil
	}

	cfg := Config{
		Workers:    2,
		BufferSize: 10,
		MaxRetries: 3,
	}

	q := New(handler, cfg)
	defer q.Shutdown(context.Background())

	// Enqueue jobs
	jobIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		jobID, err := q.Enqueue(map[string]string{"test": "data"})
		if err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}
		jobIDs[i] = jobID
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify all jobs were processed
	if got := atomic.LoadInt32(&processed); got != 5 {
		t.Errorf("Expected 5 jobs processed, got %d", got)
	}

	// Check job statuses
	for _, jobID := range jobIDs {
		job, err := q.GetJob(jobID)
		if err != nil {
			t.Errorf("Failed to get job %s: %v", jobID, err)
			continue
		}

		if job.Status != JobStatusCompleted {
			t.Errorf("Job %s status = %s, want %s", jobID, job.Status, JobStatusCompleted)
		}
	}
}

func TestQueue_FailedJobRetry(t *testing.T) {
	attempts := int32(0)

	handler := func(ctx context.Context, req interface{}) (string, error) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return "", errors.New("temporary failure")
		}
		return "msg-123", nil
	}

	cfg := Config{
		Workers:    1,
		BufferSize: 10,
		MaxRetries: 3,
	}

	q := New(handler, cfg)
	defer q.Shutdown(context.Background())

	// Enqueue job
	jobID, err := q.Enqueue(map[string]string{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for retries
	time.Sleep(5 * time.Second)

	// Verify job succeeded after retries
	job, err := q.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status != JobStatusCompleted {
		t.Errorf("Job status = %s, want %s", job.Status, JobStatusCompleted)
	}

	totalAttempts := atomic.LoadInt32(&attempts)
	if totalAttempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", totalAttempts)
	}
}

func TestQueue_PermanentFailure(t *testing.T) {
	handler := func(ctx context.Context, req interface{}) (string, error) {
		return "", errors.New("permanent failure")
	}

	cfg := Config{
		Workers:    1,
		BufferSize: 10,
		MaxRetries: 2,
	}

	q := New(handler, cfg)
	defer q.Shutdown(context.Background())

	// Enqueue job
	jobID, err := q.Enqueue(map[string]string{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for all retries
	time.Sleep(5 * time.Second)

	// Verify job failed
	job, err := q.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Status != JobStatusFailed {
		t.Errorf("Job status = %s, want %s", job.Status, JobStatusFailed)
	}

	if job.Error == nil {
		t.Error("Expected job to have error")
	}
}

func TestQueue_FullQueue(t *testing.T) {
	// Slow handler to fill queue
	handler := func(ctx context.Context, req interface{}) (string, error) {
		time.Sleep(1 * time.Second)
		return "msg-123", nil
	}

	cfg := Config{
		Workers:    1,
		BufferSize: 2,
		MaxRetries: 3,
	}

	q := New(handler, cfg)
	defer q.Shutdown(context.Background())

	// Fill the queue
	for i := 0; i < 3; i++ {
		_, err := q.Enqueue(map[string]string{"test": "data"})
		if err != nil {
			t.Logf("Queue full at iteration %d", i)
			return
		}
	}

	t.Error("Expected queue to be full")
}

func TestQueue_Stats(t *testing.T) {
	handler := func(ctx context.Context, req interface{}) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "msg-123", nil
	}

	cfg := Config{
		Workers:    2,
		BufferSize: 10,
		MaxRetries: 3,
	}

	q := New(handler, cfg)
	defer q.Shutdown(context.Background())

	// Enqueue multiple jobs
	for i := 0; i < 5; i++ {
		_, err := q.Enqueue(map[string]string{"test": "data"})
		if err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}
	}

	// Check stats immediately
	stats := q.Stats()
	if stats["total"] != 5 {
		t.Errorf("Expected 5 total jobs, got %d", stats["total"])
	}

	// Wait and check again
	time.Sleep(1 * time.Second)
	stats = q.Stats()

	if stats["completed"] != 5 {
		t.Errorf("Expected 5 completed jobs, got %d", stats["completed"])
	}
}

func TestQueue_Shutdown(t *testing.T) {
	handler := func(ctx context.Context, req interface{}) (string, error) {
		return "msg-123", nil
	}

	cfg := DefaultConfig()
	q := New(handler, cfg)

	// Enqueue some jobs
	for i := 0; i < 5; i++ {
		_, err := q.Enqueue(map[string]string{"test": "data"})
		if err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}
	}

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := q.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Try to enqueue after shutdown (should fail)
	_, err = q.Enqueue(map[string]string{"test": "data"})
	if err == nil {
		t.Error("Expected error when enqueueing after shutdown")
	}
}

func TestQueue_ConcurrentEnqueue(t *testing.T) {
	processed := int32(0)

	handler := func(ctx context.Context, req interface{}) (string, error) {
		atomic.AddInt32(&processed, 1)
		return "msg-123", nil
	}

	cfg := Config{
		Workers:    5,
		BufferSize: 50,
		MaxRetries: 3,
	}

	q := New(handler, cfg)
	defer q.Shutdown(context.Background())

	// Concurrent enqueue
	const numJobs = 20
	done := make(chan bool, numJobs)

	for i := 0; i < numJobs; i++ {
		go func() {
			_, err := q.Enqueue(map[string]string{"test": "data"})
			if err != nil {
				t.Errorf("Failed to enqueue: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all enqueues
	for i := 0; i < numJobs; i++ {
		<-done
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	if got := atomic.LoadInt32(&processed); got != numJobs {
		t.Errorf("Expected %d jobs processed, got %d", numJobs, got)
	}
}

func BenchmarkQueue_Enqueue(b *testing.B) {
	handler := func(ctx context.Context, req interface{}) (string, error) {
		return "msg-123", nil
	}

	cfg := Config{
		Workers:    10,
		BufferSize: 1000,
		MaxRetries: 3,
	}

	q := New(handler, cfg)
	defer q.Shutdown(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.Enqueue(map[string]string{"test": "data"})
	}
}
