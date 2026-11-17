package core

const sdsMaxPrealloc = 1024 * 1024 // 1 MiB — Redis SDS growth threshold

// SDS is a concept-level Simple Dynamic String: length, allocated capacity, and buffer.
// Matches Redis SDS semantics (O(1) len, preallocation on growth) without packed C headers.
type SDS struct {
	len   int
	alloc int
	buf   []byte
}

// NewSDS creates an SDS holding a copy of s. Initially len == alloc == len(s).
func NewSDS(s string) *SDS {
	n := len(s)
	buf := make([]byte, n)
	copy(buf, s)
	return &SDS{len: n, alloc: n, buf: buf}
}

// Len returns the used string length in O(1).
func (s *SDS) Len() int {
	if s == nil {
		return 0
	}
	return s.len
}

// Avail returns unused allocated space (alloc - len).
func (s *SDS) Avail() int {
	if s == nil {
		return 0
	}
	return s.alloc - s.len
}

// String returns the used payload as a Go string.
func (s *SDS) String() string {
	if s == nil || s.len == 0 {
		return ""
	}
	return string(s.buf[:s.len])
}

// Clear resets length to 0 while keeping allocated capacity.
func (s *SDS) Clear() {
	if s == nil {
		return
	}
	s.len = 0
}

// MakeRoomFor ensures at least addlen free bytes after the current len,
// using Redis SDS growth: double under 1 MiB, else newlen + 1 MiB.
func (s *SDS) MakeRoomFor(addlen int) {
	if s == nil || addlen <= 0 {
		return
	}
	if s.alloc-s.len >= addlen {
		return
	}

	newlen := s.len + addlen
	var newalloc int
	if newlen < sdsMaxPrealloc {
		newalloc = newlen * 2
	} else {
		newalloc = newlen + sdsMaxPrealloc
	}

	buf := make([]byte, newalloc)
	copy(buf, s.buf[:s.len])
	s.buf = buf
	s.alloc = newalloc
}

// Cat appends s to the SDS, growing with MakeRoomFor as needed.
func (s *SDS) Cat(str string) {
	s.CatBytes([]byte(str))
}

// CatBytes appends b to the SDS, growing with MakeRoomFor as needed.
func (s *SDS) CatBytes(b []byte) {
	if s == nil || len(b) == 0 {
		return
	}
	s.MakeRoomFor(len(b))
	copy(s.buf[s.len:], b)
	s.len += len(b)
}

// Range returns the substring from start to end inclusive (Redis GETRANGE semantics).
func (s *SDS) Range(start, end int) string {
	if s == nil || s.len == 0 {
		return ""
	}
	n := s.len
	if start < 0 {
		start = n + start
	}
	if end < 0 {
		end = n + end
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start >= n || start > end {
		return ""
	}
	if end >= n {
		end = n - 1
	}
	return string(s.buf[start : end+1])
}

// SetRange overwrites bytes starting at offset with str, zero-filling any gap.
// Returns the new length. offset must be >= 0 (caller validates).
func (s *SDS) SetRange(offset int, str string) int {
	if s == nil {
		return 0
	}
	needed := offset + len(str)
	if needed > s.len {
		oldLen := s.len
		s.MakeRoomFor(needed - s.len)
		if offset > oldLen {
			for i := oldLen; i < offset; i++ {
				s.buf[i] = 0
			}
		}
		s.len = needed
	}
	if len(str) > 0 {
		copy(s.buf[offset:], str)
	}
	return s.len
}
