package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the status of a notification job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Job represents a notification job in the queue
type Job struct {
	ID        string
	Request   interface{}
	Status    JobStatus
	Error     error
	CreatedAt time.Time
	UpdatedAt time.Time
	Retries   int
}

// Result represents the result of processing a job
type Result struct {
	JobID     string
	Success   bool
	Error     error
	MessageID string
}

// Handler is the function signature for processing jobs
type Handler func(ctx context.Context, req interface{}) (messageID string, err error)

// Queue is an in-memory job queue with goroutine workers
type Queue struct {
	jobs       chan *Job
	results    chan *Result
	jobStore   map[string]*Job
	storeMu    sync.RWMutex
	handler    Handler
	workers    int
	maxRetries int
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// Config holds queue configuration
type Config struct {
	Workers    int // Number of worker goroutines
	BufferSize int // Job queue buffer size
	MaxRetries int // Maximum retry attempts for failed jobs
}

// DefaultConfig returns default queue configuration
func DefaultConfig() Config {
	return Config{
		Workers:    10,
		BufferSize: 100,
		MaxRetries: 3,
	}
}

// New creates a new job queue
func New(handler Handler, cfg Config) *Queue {
	ctx, cancel := context.WithCancel(context.Background())

	q := &Queue{
		jobs:       make(chan *Job, cfg.BufferSize),
		results:    make(chan *Result, cfg.BufferSize),
		jobStore:   make(map[string]*Job),
		handler:    handler,
		workers:    cfg.Workers,
		maxRetries: cfg.MaxRetries,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start worker goroutines
	for i := 0; i < cfg.Workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	// Start result processor
	q.wg.Add(1)
	go q.processResults()

	return q
}

// Enqueue adds a job to the queue and returns immediately with job ID
func (q *Queue) Enqueue(req interface{}) (string, error) {
	job := &Job{
		ID:        uuid.New().String(),
		Request:   req,
		Status:    JobStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Retries:   0,
	}

	// Store job
	q.storeMu.Lock()
	q.jobStore[job.ID] = job
	q.storeMu.Unlock()

	// Send to workers
	select {
	case q.jobs <- job:
		return job.ID, nil
	case <-q.ctx.Done():
		return "", fmt.Errorf("queue is shutting down")
	default:
		return "", fmt.Errorf("queue is full")
	}
}

// GetJob returns the status of a job by ID
func (q *Queue) GetJob(jobID string) (*Job, error) {
	q.storeMu.RLock()
	defer q.storeMu.RUnlock()

	job, ok := q.jobStore[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	// Return a copy to prevent external modification
	jobCopy := *job
	return &jobCopy, nil
}

// worker processes jobs from the queue
func (q *Queue) worker(id int) {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case job, ok := <-q.jobs:
			if !ok {
				return
			}

			// Update job status
			q.updateJobStatus(job.ID, JobStatusProcessing, nil)

			// Process job with timeout
			ctx, cancel := context.WithTimeout(q.ctx, 30*time.Second)
			messageID, err := q.handler(ctx, job.Request)
			cancel()

			// Send result
			result := &Result{
				JobID:     job.ID,
				Success:   err == nil,
				Error:     err,
				MessageID: messageID,
			}

			select {
			case q.results <- result:
			case <-q.ctx.Done():
				return
			}
		}
	}
}

// processResults handles job results and retries
func (q *Queue) processResults() {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case result, ok := <-q.results:
			if !ok {
				return
			}

			if result.Success {
				q.updateJobStatus(result.JobID, JobStatusCompleted, nil)
			} else {
				// Get job for retry check
				q.storeMu.RLock()
				job, exists := q.jobStore[result.JobID]
				q.storeMu.RUnlock()

				if exists && job.Retries < q.maxRetries {
					// Retry job
					job.Retries++
					q.updateJobStatus(result.JobID, JobStatusPending, nil)

					// Re-enqueue after delay
					time.Sleep(time.Duration(job.Retries) * time.Second)
					select {
					case q.jobs <- job:
					case <-q.ctx.Done():
						return
					}
				} else {
					// Max retries reached
					q.updateJobStatus(result.JobID, JobStatusFailed, result.Error)
				}
			}
		}
	}
}

// updateJobStatus updates the status of a job
func (q *Queue) updateJobStatus(jobID string, status JobStatus, err error) {
	q.storeMu.Lock()
	defer q.storeMu.Unlock()

	if job, ok := q.jobStore[jobID]; ok {
		job.Status = status
		job.UpdatedAt = time.Now()
		if err != nil {
			job.Error = err
		}
	}
}

// Shutdown gracefully shuts down the queue
func (q *Queue) Shutdown(ctx context.Context) error {
	q.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(q.jobs)
		close(q.results)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// Stats returns queue statistics
func (q *Queue) Stats() map[string]int {
	q.storeMu.RLock()
	defer q.storeMu.RUnlock()

	stats := map[string]int{
		"total":      len(q.jobStore),
		"pending":    0,
		"processing": 0,
		"completed":  0,
		"failed":     0,
	}

	for _, job := range q.jobStore {
		switch job.Status {
		case JobStatusPending:
			stats["pending"]++
		case JobStatusProcessing:
			stats["processing"]++
		case JobStatusCompleted:
			stats["completed"]++
		case JobStatusFailed:
			stats["failed"]++
		}
	}

	return stats
}
