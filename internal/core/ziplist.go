package core

// ZipList is a concept-level ziplist: an ordered slice of string entries.
type ZipList struct {
	entries []string
}

func newZipList() *ZipList {
	return &ZipList{entries: make([]string, 0, 8)}
}

func (z *ZipList) Len() int {
	if z == nil {
		return 0
	}
	return len(z.entries)
}

func (z *ZipList) PushLeft(v string) {
	z.entries = append([]string{v}, z.entries...)
}

func (z *ZipList) PushRight(v string) {
	z.entries = append(z.entries, v)
}

func (z *ZipList) PopLeft() (string, bool) {
	if len(z.entries) == 0 {
		return "", false
	}
	v := z.entries[0]
	z.entries = z.entries[1:]
	return v, true
}

func (z *ZipList) PopRight() (string, bool) {
	if len(z.entries) == 0 {
		return "", false
	}
	last := len(z.entries) - 1
	v := z.entries[last]
	z.entries = z.entries[:last]
	return v, true
}

// Index returns the element at idx (supports negative indices). ok=false if OOB.
func (z *ZipList) Index(idx int) (string, bool) {
	n := len(z.entries)
	if idx < 0 {
		idx = n + idx
	}
	if idx < 0 || idx >= n {
		return "", false
	}
	return z.entries[idx], true
}

func (z *ZipList) Set(idx int, v string) bool {
	n := len(z.entries)
	if idx < 0 {
		idx = n + idx
	}
	if idx < 0 || idx >= n {
		return false
	}
	z.entries[idx] = v
	return true
}

// Range returns elements from start to stop inclusive (Redis LRANGE semantics).
func (z *ZipList) Range(start, stop int) []string {
	n := len(z.entries)
	if n == 0 {
		return nil
	}
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop || start >= n {
		return nil
	}
	out := make([]string, stop-start+1)
	copy(out, z.entries[start:stop+1])
	return out
}

// InsertBefore inserts v before the first occurrence of pivot. Returns false if pivot missing.
func (z *ZipList) InsertBefore(pivot, v string) bool {
	for i, e := range z.entries {
		if e == pivot {
			z.entries = append(z.entries[:i], append([]string{v}, z.entries[i:]...)...)
			return true
		}
	}
	return false
}

// InsertAfter inserts v after the first occurrence of pivot. Returns false if pivot missing.
func (z *ZipList) InsertAfter(pivot, v string) bool {
	for i, e := range z.entries {
		if e == pivot {
			z.entries = append(z.entries[:i+1], append([]string{v}, z.entries[i+1:]...)...)
			return true
		}
	}
	return false
}

// Trim keeps only elements in [start, stop] (Redis LTRIM semantics).
func (z *ZipList) Trim(start, stop int) {
	n := len(z.entries)
	if n == 0 {
		return
	}
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop || start >= n {
		z.entries = z.entries[:0]
		return
	}
	z.entries = append([]string{}, z.entries[start:stop+1]...)
}

// Rem removes count occurrences of v (Redis LREM: count>0 from head, <0 from tail, 0 all).
func (z *ZipList) Rem(count int, v string) int {
	removed := 0
	if count >= 0 {
		out := z.entries[:0:0]
		for _, e := range z.entries {
			if e == v && (count == 0 || removed < count) {
				removed++
				continue
			}
			out = append(out, e)
		}
		z.entries = out
		return removed
	}
	// count < 0: remove |count| from tail
	limit := -count
	out := make([]string, 0, len(z.entries))
	for i := len(z.entries) - 1; i >= 0; i-- {
		e := z.entries[i]
		if e == v && removed < limit {
			removed++
			continue
		}
		out = append(out, e)
	}
	// reverse back
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	z.entries = out
	return removed
}

func (z *ZipList) Entries() []string {
	if z == nil {
		return nil
	}
	out := make([]string, len(z.entries))
	copy(out, z.entries)
	return out
}

func (z *ZipList) maxEntryLen() int {
	max := 0
	for _, e := range z.entries {
		if len(e) > max {
			max = len(e)
		}
	}
	return max
}

func ziplistFromEntries(entries []string) *ZipList {
	z := newZipList()
	z.entries = append(z.entries, entries...)
	return z
}
