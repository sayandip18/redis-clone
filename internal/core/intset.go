package core

import (
	"sort"
	"strconv"
)

// IntSet is a concept-level intset: sorted unique int64 values.
type IntSet struct {
	nums []int64
}

func newIntSet() *IntSet {
	return &IntSet{nums: make([]int64, 0, 8)}
}

func (s *IntSet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.nums)
}

func (s *IntSet) Contains(n int64) bool {
	i := sort.Search(len(s.nums), func(i int) bool { return s.nums[i] >= n })
	return i < len(s.nums) && s.nums[i] == n
}

// Add inserts n if absent. Returns true if newly added.
func (s *IntSet) Add(n int64) bool {
	i := sort.Search(len(s.nums), func(i int) bool { return s.nums[i] >= n })
	if i < len(s.nums) && s.nums[i] == n {
		return false
	}
	s.nums = append(s.nums, 0)
	copy(s.nums[i+1:], s.nums[i:])
	s.nums[i] = n
	return true
}

// Remove deletes n if present. Returns true if removed.
func (s *IntSet) Remove(n int64) bool {
	i := sort.Search(len(s.nums), func(i int) bool { return s.nums[i] >= n })
	if i >= len(s.nums) || s.nums[i] != n {
		return false
	}
	s.nums = append(s.nums[:i], s.nums[i+1:]...)
	return true
}

func (s *IntSet) Members() []string {
	out := make([]string, len(s.nums))
	for i, n := range s.nums {
		out[i] = strconv.FormatInt(n, 10)
	}
	return out
}

func (s *IntSet) toHashtable() map[string]struct{} {
	m := make(map[string]struct{}, len(s.nums))
	for _, n := range s.nums {
		m[strconv.FormatInt(n, 10)] = struct{}{}
	}
	return m
}

func parseIntMember(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	// Redis intset rejects leading zeros / plus signs inconsistently;
	// require canonical decimal form for int membership.
	if strconv.FormatInt(n, 10) != s {
		return 0, false
	}
	return n, true
}
