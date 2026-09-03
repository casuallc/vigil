/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package poll

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"
)

// Task statuses reported back to upstreams.
const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusTimeout = "timeout"
)

// Task is the unit of work pulled from an upstream. The topic is assigned
// by the upstream and used by bbx only as the serialization key: tasks of
// the same topic run strictly one after another, different topics run in
// parallel.
type Task struct {
	ID         string          `json:"id"`
	Topic      string          `json:"topic"`
	TimeoutSec int             `json:"timeout_sec"`
	Action     json.RawMessage `json:"action"`
	AckURL     string          `json:"ack_url,omitempty"`
	CreatedAt  string          `json:"created_at,omitempty"`

	// acker posts the result back to the upstream this task came from.
	acker Acker
	// onComplete is invoked after ack with the final payload (dedup feed).
	onComplete func(id string, p AckPayload)
}

// AckPayload is posted back to the upstream that produced the task.
type AckPayload struct {
	ID         string          `json:"id"`
	Status     string          `json:"status"` // success | failed | timeout
	Result     json.RawMessage `json:"result,omitempty"`
	ExitCode   int             `json:"exit_code"`
	DurationMs int64           `json:"duration_ms"`
	Error      string          `json:"error,omitempty"`
}

// Acker posts a task result back to its origin upstream.
type Acker interface {
	Ack(ctx context.Context, payload AckPayload) error
}

// errShuttingDown is returned by Dispatch once the dispatcher is stopping.
var errShuttingDown = errors.New("poll: dispatcher shutting down")

// taskExecutor abstracts task execution so the dispatcher can be tested
// with fake executors.
type taskExecutor interface {
	Execute(ctx context.Context, t *Task) (json.RawMessage, error)
}

// Dispatcher routes tasks to per-topic queues. Each queue has exactly one
// worker goroutine, giving serial execution within a topic and parallel
// execution across topics. Queues are created lazily, bounded, and reaped
// after an idle TTL.
type Dispatcher struct {
	executor     taskExecutor
	taskTimeout  time.Duration
	queueBuf     int
	maxTopics    int
	idleTTL      time.Duration
	drainTimeout time.Duration

	mu     sync.Mutex
	queues map[string]*topicQueue

	done chan struct{}
	wg   sync.WaitGroup

	stopOnce sync.Once
}

func newDispatcher(ex taskExecutor, d *Defaults) *Dispatcher {
	return &Dispatcher{
		executor:     ex,
		taskTimeout:  d.TaskTimeout.Std(),
		queueBuf:     d.QueueBuffer,
		maxTopics:    d.MaxTopics,
		idleTTL:      d.TopicIdleTTL.Std(),
		drainTimeout: drainTimeout,
		queues:       make(map[string]*topicQueue),
		done:         make(chan struct{}),
	}
}

// Dispatch routes a task to its topic queue. It blocks when the queue is
// full, which propagates backpressure up to the poller (the poller loop is
// "pull a batch -> dispatch all -> pull again"), so tasks are never lost.
func (d *Dispatcher) Dispatch(t *Task) error {
	d.mu.Lock()
	q := d.queueForLocked(t.Topic)
	q.pending++
	d.mu.Unlock()

	err := q.enqueue(t)

	d.mu.Lock()
	q.pending--
	d.mu.Unlock()
	return err
}

// queueForLocked returns the queue for a topic, creating it (and starting
// its worker) on first sight. When the number of live topic queues reaches
// maxTopics, unknown topics fall back to the shared "default" queue so
// tasks are never dropped.
func (d *Dispatcher) queueForLocked(topic string) *topicQueue {
	if topic == "" {
		topic = "default"
	}
	if q, ok := d.queues[topic]; ok {
		return q
	}
	if topic != "default" && len(d.queues) >= d.maxTopics {
		if q, ok := d.queues["default"]; ok {
			return q
		}
		topic = "default"
	}
	q := &topicQueue{
		topic: topic,
		ch:    make(chan *Task, d.queueBuf),
		d:     d,
	}
	d.queues[topic] = q
	d.wg.Add(1)
	go q.run()
	return q
}

// Shutdown stops accepting new tasks, lets workers drain their queues for
// drainTimeout, NACKs whatever is left, and waits for all workers to exit.
func (d *Dispatcher) Shutdown() {
	d.stopOnce.Do(func() { close(d.done) })
	d.wg.Wait()
}

// executeAndAck runs one task to completion and posts the result back.
func (d *Dispatcher) executeAndAck(t *Task) {
	start := time.Now()
	timeout := d.taskTimeout
	if t.TimeoutSec > 0 {
		timeout = time.Duration(t.TimeoutSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := d.executor.Execute(ctx, t)

	payload := AckPayload{ID: t.ID, DurationMs: time.Since(start).Milliseconds()}
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		payload.Status = StatusTimeout
		payload.ExitCode = 124
		payload.Error = "task timeout"
	case err != nil:
		payload.Status = StatusFailed
		payload.ExitCode = 1
		payload.Error = err.Error()
	default:
		payload.Status = StatusSuccess
		payload.ExitCode = 0
		payload.Result = result
	}

	d.ack(t, payload)
}

// nack reports a task as failed without executing it (used on shutdown).
func (d *Dispatcher) nack(t *Task, reason string) {
	payload := AckPayload{
		ID:       t.ID,
		Status:   StatusFailed,
		ExitCode: 1,
		Error:    reason,
	}
	d.ack(t, payload)
}

func (d *Dispatcher) ack(t *Task, payload AckPayload) {
	if t.acker == nil {
		log.Printf("[poll] task %s has no acker, dropping result", t.ID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := t.acker.Ack(ctx, payload); err != nil {
		// The upstream will re-dispatch after its own timeout; the dedup
		// cache (fed via onComplete) then answers with this payload.
		log.Printf("[poll] ack task %s failed: %v", t.ID, err)
	}
	if t.onComplete != nil {
		t.onComplete(t.ID, payload)
	}
}

// topicQueue is a bounded queue with exactly one worker goroutine.
type topicQueue struct {
	topic string
	ch    chan *Task
	d     *Dispatcher

	// pending counts dispatchers that hold a reference to this queue and
	// are about to (blocking) enqueue; it prevents reaping races.
	pending int // guarded by d.mu
}

// enqueue blocks until the task is queued or the dispatcher shuts down.
func (q *topicQueue) enqueue(t *Task) error {
	select {
	case q.ch <- t:
		return nil
	case <-q.d.done:
		return errShuttingDown
	}
}

// run is the worker loop: take one task, execute it (including ack), then
// take the next. The worker exits when the queue has been empty and idle
// for idleTTL, or when the dispatcher shuts down.
func (q *topicQueue) run() {
	defer q.d.wg.Done()
	idle := time.NewTimer(q.d.idleTTL)
	defer idle.Stop()

	for {
		select {
		case t := <-q.ch:
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			q.d.executeAndAck(t)
			idle.Reset(q.d.idleTTL)
		case <-idle.C:
			q.d.mu.Lock()
			if len(q.ch) == 0 && q.pending == 0 {
				delete(q.d.queues, q.topic)
				q.d.mu.Unlock()
				return
			}
			q.d.mu.Unlock()
			idle.Reset(q.d.idleTTL)
		case <-q.d.done:
			q.drain()
			return
		}
	}
}

// drain executes queued tasks until the queue is empty or drainTimeout
// elapses, then NACKs whatever is left.
func (q *topicQueue) drain() {
	deadline := time.After(q.d.drainTimeout)
	for {
		select {
		case t := <-q.ch:
			q.d.executeAndAck(t)
		default:
			return // queue empty
		case <-deadline:
			for {
				select {
				case t := <-q.ch:
					q.d.nack(t, "agent shutdown")
				default:
					return
				}
			}
		}
	}
}
