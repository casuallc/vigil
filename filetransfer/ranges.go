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

// interval is a half-open byte range [start, end).
type interval struct{ start, end int64 }

// intervalSet is a sorted set of disjoint, coalesced byte intervals. The RECV
// side uses it to track which parts of a file have landed, so chunks may
// arrive out of order (parallel KAFKA sends) and completion is detected by
// coverage rather than arrival sequence.
type intervalSet struct{ intervals []interval }

// insert adds [start, end) to the set, coalescing overlaps, and returns the
// number of newly covered bytes.
func (s *intervalSet) insert(start, end int64) int64 {
	if end <= start {
		return 0
	}
	added := end - start
	// A fresh slice is required: writing into s.intervals[:0] while ranging
	// over s.intervals corrupts unread elements whenever one iteration
	// appends two entries (insert-before-first with further intervals
	// pending), which silently drops ranges and can double-discount added.
	out := make([]interval, 0, len(s.intervals)+1)
	inserted := false
	for _, iv := range s.intervals {
		if iv.end < start {
			out = append(out, iv)
			continue
		}
		if iv.start > end {
			if !inserted {
				out = append(out, interval{start, end})
				inserted = true
			}
			out = append(out, iv)
			continue
		}
		// Overlapping (or touching) interval: discount the already-covered
		// part and extend the pending range to swallow iv.
		ovStart, ovEnd := start, end
		if iv.start > ovStart {
			ovStart = iv.start
		}
		if iv.end < ovEnd {
			ovEnd = iv.end
		}
		added -= ovEnd - ovStart
		if iv.start < start {
			start = iv.start
		}
		if iv.end > end {
			end = iv.end
		}
	}
	if !inserted {
		out = append(out, interval{start, end})
	}
	if added < 0 {
		added = 0 // defensive: disjoint intervals can never double-discount
	}
	s.intervals = out
	return added
}

// covered returns the total number of bytes across all intervals.
func (s *intervalSet) covered() int64 {
	var n int64
	for _, iv := range s.intervals {
		n += iv.end - iv.start
	}
	return n
}
