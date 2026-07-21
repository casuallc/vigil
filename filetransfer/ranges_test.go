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

import "testing"

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
