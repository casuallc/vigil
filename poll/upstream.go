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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// pollResponse is the upstream's answer to a long-poll request.
type pollResponse struct {
	Tasks   []Task `json:"tasks"`
	HasMore bool   `json:"has_more"`
}

// Options carries the server-side wiring the agent needs for local
// loopback calls (api / ws_bridge task types).
type Options struct {
	// InternalToken authenticates loopback requests towards bbx's own API.
	InternalToken string
	// LoopbackAddr is the host:port of the local API server (e.g. 127.0.0.1:57575).
	LoopbackAddr string
	// LoopbackTLS indicates the local API server speaks HTTPS.
	LoopbackTLS bool
}

// Agent is the poll-mode entry point: it owns the dispatcher and one
// poller goroutine per configured upstream.
type Agent struct {
	cfg        *PollConfig
	agentID    string
	dispatcher *Dispatcher
	executor   *Executor
	pollers    []*poller

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewAgent builds the agent from configuration. It returns (nil, nil) when
// poll mode is disabled.
func NewAgent(cfg *PollConfig, opts Options) (*Agent, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	agentID := cfg.AgentID
	if agentID == "" {
		hostname, err := os.Hostname()
		if err != nil || hostname == "" {
			hostname = "bbx-agent"
		}
		agentID = hostname
	}

	executor := newExecutor(opts, cfg.Defaults.TaskTimeout.Std())
	dispatcher := newDispatcher(executor, &cfg.Defaults)

	a := &Agent{
		cfg:        cfg,
		agentID:    agentID,
		dispatcher: dispatcher,
		executor:   executor,
	}
	for i := range cfg.Upstreams {
		p, err := newPoller(&cfg.Upstreams[i], cfg, agentID, dispatcher)
		if err != nil {
			return nil, err
		}
		a.pollers = append(a.pollers, p)
	}
	return a, nil
}

// Start launches one poller goroutine per upstream.
func (a *Agent) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	for _, p := range a.pollers {
		a.wg.Add(1)
		go func(p *poller) {
			defer a.wg.Done()
			p.run(ctx)
		}(p)
	}
	log.Printf("[poll] agent %q started with %d upstream(s)", a.agentID, len(a.pollers))
}

// Stop cancels all pollers and drains the dispatcher gracefully.
func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
	a.dispatcher.Shutdown()
	log.Printf("[poll] agent stopped")
}

// poller is the per-upstream polling loop with its own busy/idle state
// machine, backoff, authentication and TLS configuration.
type poller struct {
	name       string
	cfg        *UpstreamConfig
	defaults   *Defaults
	agentID    string
	client     *http.Client
	dispatcher *Dispatcher
	dedup      *dedupCache
}

func newPoller(cfg *UpstreamConfig, root *PollConfig, agentID string, d *Dispatcher) (*poller, error) {
	tlsCfg, err := buildTLSConfig(root.tlsConfigFor(cfg))
	if err != nil {
		return nil, fmt.Errorf("poll: upstream %q: %w", cfg.Name, err)
	}
	// The client timeout must exceed the long-poll hold.
	timeout := root.Defaults.LongPollWait.Std() + 15*time.Second
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
	return &poller{
		name:       cfg.Name,
		cfg:        cfg,
		defaults:   &root.Defaults,
		agentID:    agentID,
		client:     client,
		dispatcher: d,
		dedup:      newDedupCache(dedupCacheSize),
	}, nil
}

func buildTLSConfig(c TLSConfig) (*tls.Config, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify}
	if c.CAFile != "" {
		pem, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read ca_file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("ca_file %s contains no PEM certificates", c.CAFile)
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}

// run is the busy/idle state machine:
//
//	IDLE: long-poll with wait=long_poll_wait (the hold doubles as keepalive);
//	      empty polls back off gently (2s doubling up to idle_backoff_max).
//	BUSY: short polls every busy_interval; has_more=true polls immediately;
//	      busy_to_idle_empty_polls consecutive empty polls drop back to IDLE.
//
// Connection errors back off exponentially (capped at idle_backoff_max) and
// never affect other upstreams' pollers.
func (p *poller) run(ctx context.Context) {
	busy := false
	emptyStreak := 0
	errBackoff := time.Second
	idleBackoff := 2 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		wait := p.defaults.LongPollWait.Std()
		if busy {
			wait = 0
		}
		tasks, hasMore, err := p.pollOnce(ctx, wait)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[poll] upstream %s poll error: %v (retry in %s)", p.name, err, errBackoff)
			if !sleepCtx(ctx, errBackoff) {
				return
			}
			errBackoff *= 2
			if errBackoff > p.defaults.IdleBackoffMax.Std() {
				errBackoff = p.defaults.IdleBackoffMax.Std()
			}
			continue
		}
		errBackoff = time.Second

		if len(tasks) > 0 || hasMore {
			busy = true
			emptyStreak = 0
		} else {
			emptyStreak++
			if busy && emptyStreak >= p.defaults.BusyToIdleEmptyPolls {
				busy = false
				emptyStreak = 0
				idleBackoff = 2 * time.Second
			}
		}

		p.dispatchAll(tasks)

		if hasMore {
			continue // pull again immediately
		}
		if busy {
			if !sleepCtx(ctx, p.defaults.BusyInterval.Std()) {
				return
			}
		} else if len(tasks) == 0 {
			if !sleepCtx(ctx, idleBackoff) {
				return
			}
			idleBackoff *= 2
			if idleBackoff > p.defaults.IdleBackoffMax.Std() {
				idleBackoff = p.defaults.IdleBackoffMax.Std()
			}
		}
	}
}

// dispatchAll answers idempotent duplicates from the dedup cache and hands
// the rest to the dispatcher. A full queue blocks here, which pauses
// polling and propagates backpressure to the upstream.
func (p *poller) dispatchAll(tasks []Task) {
	for i := range tasks {
		t := &tasks[i]
		if payload, ok := p.dedup.Get(t.ID); ok {
			// Re-dispatched task we already finished: re-ack, don't re-run.
			acked := payload
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := p.ackTask(ctx, t.AckURL, acked); err != nil {
					log.Printf("[poll] re-ack task %s failed: %v", t.ID, err)
				}
			}()
			continue
		}
		t.acker = &taskAcker{p: p, ackURL: t.AckURL}
		t.onComplete = func(id string, payload AckPayload) {
			p.dedup.Put(id, payload)
		}
		if err := p.dispatcher.Dispatch(t); err != nil {
			// Dispatcher is shutting down.
			return
		}
	}
}

// pollOnce issues one (long-)poll request against the upstream.
func (p *poller) pollOnce(ctx context.Context, wait time.Duration) ([]Task, bool, error) {
	u := fmt.Sprintf("%s/poll?agent=%s&wait=%s",
		p.cfg.Endpoint, url.QueryEscape(p.agentID), url.QueryEscape(wait.String()))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, false, err
	}
	p.setAuth(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("poll returned status %d", resp.StatusCode)
	}
	var pr pollResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, false, fmt.Errorf("decode poll response: %w", err)
	}
	return pr.Tasks, pr.HasMore, nil
}

// taskAcker posts the result to the upstream the task was pulled from,
// honoring the task-level ack_url override.
type taskAcker struct {
	p      *poller
	ackURL string
}

func (a *taskAcker) Ack(ctx context.Context, payload AckPayload) error {
	return a.p.ackTask(ctx, a.ackURL, payload)
}

func (p *poller) ackTask(ctx context.Context, ackURL string, payload AckPayload) error {
	endpoint := p.cfg.Endpoint + "/ack"
	if ackURL != "" {
		endpoint = ackURL
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	p.setAuth(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ack returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *poller) setAuth(req *http.Request) {
	if p.cfg.Auth.Username != "" || p.cfg.Auth.Password != "" {
		req.SetBasicAuth(p.cfg.Auth.Username, p.cfg.Auth.Password)
	}
}

// sleepCtx sleeps for d or until ctx is canceled; it returns false when
// the context was canceled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
