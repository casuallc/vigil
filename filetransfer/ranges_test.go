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

package filetransfer

import (
	"math/rand"
	"testing"
)

func TestIntervalSetDisjointInsertAndCoalesce(t *testing.T) {
	var s intervalSet
	if added := s.insert(10, 20); added != 10 {
		t.Fatalf("insert [10,20): added=%d, want 10", added)
	}
	if added := s.insert(0, 5); added != 5 {
		t.Fatalf("insert [0,5): added=%d, want 5", added)
	}
	if got := s.covered(); got != 15 {
		t.Fatalf("covered=%d, want 15", got)
	}
	// Bridge the gap: merges both intervals into one.
	if added := s.insert(5, 10); added != 5 {
		t.Fatalf("insert [5,10): added=%d, want 5", added)
	}
	if got := s.covered(); got != 20 {
		t.Fatalf("covered=%d, want 20", got)
	}
	if len(s.intervals) != 1 || s.intervals[0] != (interval{0, 20}) {
		t.Fatalf("expected single interval [0,20), got %+v", s.intervals)
	}
}

func TestIntervalSetOverlapCountsOnlyNewBytes(t *testing.T) {
	var s intervalSet
	s.insert(0, 100)
	if added := s.insert(50, 60); added != 0 {
		t.Fatalf("fully covered insert: added=%d, want 0", added)
	}
	if added := s.insert(90, 110); added != 10 {
		t.Fatalf("partial overlap: added=%d, want 10", added)
	}
	if got := s.covered(); got != 110 {
		t.Fatalf("covered=%d, want 110", got)
	}
}

func TestIntervalSetOutOfOrderShuffledChunks(t *testing.T) {
	// Simulate a 10-chunk file delivered in reverse order.
	var s intervalSet
	const chunk = 100
	for i := 9; i >= 0; i-- {
		s.insert(int64(i*chunk), int64((i+1)*chunk))
	}
	if got := s.covered(); got != 10*chunk {
		t.Fatalf("covered=%d, want %d", got, 10*chunk)
	}
	if len(s.intervals) != 1 {
		t.Fatalf("expected coalesced single interval, got %+v", s.intervals)
	}
}

// TestIntervalSetInsertBeforeFirstWithPendingIntervals is a regression test
// for the aliasing bug where out reused s.intervals' backing array: an
// insert before the first interval appends two entries in one iteration and
// used to overwrite unread intervals, silently dropping ranges.
func TestIntervalSetInsertBeforeFirstWithPendingIntervals(t *testing.T) {
	var s intervalSet
	s.insert(100, 200)
	s.insert(300, 400)
	s.insert(500, 600)
	s.insert(700, 800)

	if added := s.insert(0, 50); added != 50 {
		t.Fatalf("insert before first: added=%d, want 50", added)
	}
	want := []interval{{0, 50}, {100, 200}, {300, 400}, {500, 600}, {700, 800}}
	if len(s.intervals) != len(want) {
		t.Fatalf("intervals=%+v, want %+v", s.intervals, want)
	}
	for i, iv := range want {
		if s.intervals[i] != iv {
			t.Fatalf("intervals=%+v, want %+v", s.intervals, want)
		}
	}
	if got := s.covered(); got != 450 {
		t.Fatalf("covered=%d, want 450", got)
	}
}

// TestIntervalSetRandomShufflesMatchesModel stress-tests insert against a
// simple boolean-array model over many deterministic shuffles, including
// duplicate (re-delivered) chunks and coalesced multi-chunk intervals.
func TestIntervalSetRandomShufflesMatchesModel(t *testing.T) {
	const (
		chunks    = 64
		chunkSize = 100
		trials    = 50
	)
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < trials; trial++ {
		var s intervalSet
		model := make([]bool, chunks*chunkSize)
		order := rng.Perm(chunks)
		// Re-deliver some chunks (duplicates must be idempotent).
		for _, dup := range order[:chunks/4] {
			order = append(order, dup)
		}
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

		var addedTotal int64
		for _, c := range order {
			start := int64(c * chunkSize)
			end := start + chunkSize
			added := s.insert(start, end)
			if added < 0 {
				t.Fatalf("trial %d: negative added %d", trial, added)
			}
			addedTotal += added
			var modelAdded int64
			for b := start; b < end; b++ {
				if !model[b] {
					model[b] = true
					modelAdded++
				}
			}
			if added != modelAdded {
				t.Fatalf("trial %d chunk %d: added=%d, model=%d", trial, c, added, modelAdded)
			}
		}
		if got := s.covered(); got != chunks*chunkSize {
			t.Fatalf("trial %d: covered=%d, want %d (intervals=%+v)", trial, got, chunks*chunkSize, s.intervals)
		}
		if addedTotal != chunks*chunkSize {
			t.Fatalf("trial %d: sum(added)=%d, want %d", trial, addedTotal, chunks*chunkSize)
		}
		if len(s.intervals) != 1 {
			t.Fatalf("trial %d: expected coalesced single interval, got %+v", trial, s.intervals)
		}
	}
}

func TestIntervalSetEmptyAndInvalidRanges(t *testing.T) {
	var s intervalSet
	if added := s.insert(5, 5); added != 0 {
		t.Fatalf("empty range: added=%d, want 0", added)
	}
	if added := s.insert(9, 4); added != 0 {
		t.Fatalf("inverted range: added=%d, want 0", added)
	}
	if got := s.covered(); got != 0 {
		t.Fatalf("covered=%d, want 0", got)
	}
}
