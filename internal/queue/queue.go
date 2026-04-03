package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/MathiasDPX/archivetube/internal/metrics"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusError      Status = "error"
)

type Job struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Quality   string    `json:"quality"`
	Status    Status    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ArchiveFunc func(ctx context.Context, url string, quality string) error

type Queue struct {
	mu      sync.Mutex
	jobs    []*Job
	nextID  int
	archive ArchiveFunc
}

func New(archiveFn ArchiveFunc) *Queue {
	q := &Queue{
		archive: archiveFn,
	}
	go q.worker()
	metrics.SetQueueSize(0)
	return q
}

func (q *Queue) Enqueue(url string, quality string) *Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.nextID++
	job := &Job{
		ID:        fmt.Sprintf("%d", q.nextID),
		URL:       url,
		Quality:   quality,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
	q.jobs = append(q.jobs, job)
	q.updateMetrics()
	return job
}

func (q *Queue) Jobs() []Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	out := make([]Job, len(q.jobs))
	for i, j := range q.jobs {
		out[i] = *j
	}
	return out
}

func (q *Queue) RemoveJob(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, j := range q.jobs {
		if j.ID == id && (j.Status == StatusDone || j.Status == StatusError) {
			q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
			return true
		}
	}
	return false
}

func (q *Queue) ClearFinished() {
	q.mu.Lock()
	defer q.mu.Unlock()

	var remaining []*Job
	for _, j := range q.jobs {
		if j.Status == StatusPending || j.Status == StatusProcessing {
			remaining = append(remaining, j)
		}
	}
	q.jobs = remaining
	q.updateMetrics()
}

func (q *Queue) worker() {
	for {
		job := q.nextPending()
		if job == nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		q.setStatus(job.ID, StatusProcessing, "")
		log.Printf("queue: processing %s (%s)", job.ID, job.URL)

		err := q.archive(context.Background(), job.URL, job.Quality)
		if err != nil {
			log.Printf("queue: error for %s: %v", job.ID, err)
			q.setStatus(job.ID, StatusError, err.Error())
		} else {
			log.Printf("queue: done %s", job.ID)
			q.setStatus(job.ID, StatusDone, "")
		}
	}
}

func (q *Queue) nextPending() *Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, j := range q.jobs {
		if j.Status == StatusPending {
			return j
		}
	}
	return nil
}

func (q *Queue) setStatus(id string, status Status, errMsg string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, j := range q.jobs {
		if j.ID == id {
			j.Status = status
			j.Error = errMsg
			break
		}
	}
	q.updateMetrics()
}

// updateMetrics sets the queue size gauge. Must be called with q.mu held.
func (q *Queue) updateMetrics() {
	n := 0
	for _, j := range q.jobs {
		if j.Status == StatusPending || j.Status == StatusProcessing {
			n++
		}
	}
	metrics.SetQueueSize(n)
}
