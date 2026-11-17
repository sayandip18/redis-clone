package core

import "github.com/sayandip18/redis-clone/config"

// qlNode is one quicklist node holding a ZipList fill container.
type qlNode struct {
	zl   *ZipList
	prev *qlNode
	next *qlNode
}

// QuickList is a concept-level quicklist: doubly-linked ZipList nodes.
type QuickList struct {
	head *qlNode
	tail *qlNode
	len  int // total elements
	fill int // max entries per node
}

func newQuickList() *QuickList {
	fill := config.ListMaxZiplistEntries
	if fill <= 0 {
		fill = 512
	}
	return &QuickList{fill: fill}
}

func quicklistFromZiplist(z *ZipList) *QuickList {
	ql := newQuickList()
	if z == nil || z.Len() == 0 {
		return ql
	}
	// Split into fill-sized nodes.
	entries := z.Entries()
	for i := 0; i < len(entries); {
		end := i + ql.fill
		if end > len(entries) {
			end = len(entries)
		}
		node := &qlNode{zl: ziplistFromEntries(entries[i:end])}
		ql.appendNode(node)
		i = end
	}
	ql.len = len(entries)
	return ql
}

func (ql *QuickList) appendNode(n *qlNode) {
	if ql.head == nil {
		ql.head = n
		ql.tail = n
		return
	}
	n.prev = ql.tail
	ql.tail.next = n
	ql.tail = n
}

func (ql *QuickList) prependNode(n *qlNode) {
	if ql.head == nil {
		ql.head = n
		ql.tail = n
		return
	}
	n.next = ql.head
	ql.head.prev = n
	ql.head = n
}

func (ql *QuickList) Len() int {
	if ql == nil {
		return 0
	}
	return ql.len
}

func (ql *QuickList) ensureHead() *qlNode {
	if ql.head == nil {
		n := &qlNode{zl: newZipList()}
		ql.head = n
		ql.tail = n
	}
	return ql.head
}

func (ql *QuickList) ensureTail() *qlNode {
	if ql.tail == nil {
		return ql.ensureHead()
	}
	return ql.tail
}

func (ql *QuickList) PushLeft(v string) {
	head := ql.ensureHead()
	if head.zl.Len() >= ql.fill {
		n := &qlNode{zl: newZipList()}
		ql.prependNode(n)
		head = n
	}
	head.zl.PushLeft(v)
	ql.len++
}

func (ql *QuickList) PushRight(v string) {
	tail := ql.ensureTail()
	if tail.zl.Len() >= ql.fill {
		n := &qlNode{zl: newZipList()}
		ql.appendNode(n)
		tail = n
	}
	tail.zl.PushRight(v)
	ql.len++
}

func (ql *QuickList) PopLeft() (string, bool) {
	if ql.len == 0 || ql.head == nil {
		return "", false
	}
	v, ok := ql.head.zl.PopLeft()
	if !ok {
		return "", false
	}
	ql.len--
	if ql.head.zl.Len() == 0 {
		ql.removeNode(ql.head)
	}
	return v, true
}

func (ql *QuickList) PopRight() (string, bool) {
	if ql.len == 0 || ql.tail == nil {
		return "", false
	}
	v, ok := ql.tail.zl.PopRight()
	if !ok {
		return "", false
	}
	ql.len--
	if ql.tail != nil && ql.tail.zl.Len() == 0 {
		ql.removeNode(ql.tail)
	}
	return v, true
}

func (ql *QuickList) removeNode(n *qlNode) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		ql.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		ql.tail = n.prev
	}
}

// nodeAt walks to the node/offset containing absolute index idx (non-negative).
func (ql *QuickList) nodeAt(idx int) (*qlNode, int) {
	if idx < 0 || idx >= ql.len {
		return nil, -1
	}
	n := ql.head
	for n != nil {
		l := n.zl.Len()
		if idx < l {
			return n, idx
		}
		idx -= l
		n = n.next
	}
	return nil, -1
}

func (ql *QuickList) resolveIndex(idx int) int {
	if idx < 0 {
		idx = ql.len + idx
	}
	return idx
}

func (ql *QuickList) Index(idx int) (string, bool) {
	idx = ql.resolveIndex(idx)
	n, off := ql.nodeAt(idx)
	if n == nil {
		return "", false
	}
	return n.zl.Index(off)
}

func (ql *QuickList) Set(idx int, v string) bool {
	idx = ql.resolveIndex(idx)
	n, off := ql.nodeAt(idx)
	if n == nil {
		return false
	}
	return n.zl.Set(off, v)
}

func (ql *QuickList) Entries() []string {
	if ql == nil || ql.len == 0 {
		return nil
	}
	out := make([]string, 0, ql.len)
	for n := ql.head; n != nil; n = n.next {
		out = append(out, n.zl.Entries()...)
	}
	return out
}

func (ql *QuickList) Range(start, stop int) []string {
	entries := ql.Entries()
	z := ziplistFromEntries(entries)
	return z.Range(start, stop)
}

func (ql *QuickList) InsertBefore(pivot, v string) bool {
	for n := ql.head; n != nil; n = n.next {
		if n.zl.InsertBefore(pivot, v) {
			ql.len++
			ql.maybeSplit(n)
			return true
		}
	}
	return false
}

func (ql *QuickList) InsertAfter(pivot, v string) bool {
	for n := ql.head; n != nil; n = n.next {
		if n.zl.InsertAfter(pivot, v) {
			ql.len++
			ql.maybeSplit(n)
			return true
		}
	}
	return false
}

func (ql *QuickList) maybeSplit(n *qlNode) {
	if n.zl.Len() <= ql.fill {
		return
	}
	entries := n.zl.Entries()
	mid := len(entries) / 2
	n.zl = ziplistFromEntries(entries[:mid])
	right := &qlNode{zl: ziplistFromEntries(entries[mid:])}
	right.prev = n
	right.next = n.next
	if n.next != nil {
		n.next.prev = right
	} else {
		ql.tail = right
	}
	n.next = right
}

func (ql *QuickList) Trim(start, stop int) {
	entries := ql.Entries()
	z := ziplistFromEntries(entries)
	z.Trim(start, stop)
	kept := z.Entries()
	ql.head = nil
	ql.tail = nil
	ql.len = 0
	if len(kept) == 0 {
		return
	}
	for i := 0; i < len(kept); {
		end := i + ql.fill
		if end > len(kept) {
			end = len(kept)
		}
		node := &qlNode{zl: ziplistFromEntries(kept[i:end])}
		ql.appendNode(node)
		i = end
	}
	ql.len = len(kept)
}

func (ql *QuickList) Rem(count int, v string) int {
	entries := ql.Entries()
	z := ziplistFromEntries(entries)
	removed := z.Rem(count, v)
	ql.rebuild(z.Entries())
	return removed
}

func (ql *QuickList) rebuild(entries []string) {
	ql.head = nil
	ql.tail = nil
	ql.len = 0
	if len(entries) == 0 {
		return
	}
	for i := 0; i < len(entries); {
		end := i + ql.fill
		if end > len(entries) {
			end = len(entries)
		}
		node := &qlNode{zl: ziplistFromEntries(entries[i:end])}
		ql.appendNode(node)
		i = end
	}
	ql.len = len(entries)
}
