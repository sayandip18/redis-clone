package core

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

// BlockWaiter is a client waiting on one or more list keys.
type BlockWaiter struct {
	Client   *Client
	Writer   io.Writer
	Keys     []string
	Deadline time.Time // zero means block forever
	Left     bool
}

var blockWaiters = make(map[string][]*BlockWaiter) // key → FIFO waiters

// ClientBlockCleanup removes a disconnected client from all waiter queues.
func ClientBlockCleanup(c *Client) {
	if c == nil {
		return
	}
	removeWaiter(c)
}

func removeWaiter(c *Client) {
	for key, list := range blockWaiters {
		out := list[:0]
		for _, w := range list {
			if w.Client != c {
				out = append(out, w)
			}
		}
		if len(out) == 0 {
			delete(blockWaiters, key)
		} else {
			blockWaiters[key] = out
		}
	}
}

func registerWaiter(w *BlockWaiter) {
	for _, key := range w.Keys {
		blockWaiters[key] = append(blockWaiters[key], w)
	}
}

func unregisterWaiter(w *BlockWaiter) {
	for _, key := range w.Keys {
		list := blockWaiters[key]
		out := list[:0]
		for _, x := range list {
			if x != w {
				out = append(out, x)
			}
		}
		if len(out) == 0 {
			delete(blockWaiters, key)
		} else {
			blockWaiters[key] = out
		}
	}
}

// tryUnblockOnPush wakes waiters for key after a successful push (one element per waiter).
func tryUnblockOnPush(key string) {
	for {
		list := blockWaiters[key]
		if len(list) == 0 {
			return
		}
		w := list[0]
		side := listRight
		if w.Left {
			side = listLeft
		}

		obj := Get(key)
		if obj == nil || obj.Type != ObjList {
			return
		}
		c, err := getListContainer(obj)
		if err != nil || c.Len() == 0 {
			return
		}
		var (
			v  string
			ok bool
		)
		if side == listLeft {
			v, ok = c.PopLeft()
		} else {
			v, ok = c.PopRight()
		}
		if !ok {
			return
		}
		if c.Len() == 0 {
			deleteKey(key)
		} else {
			bumpVersion(key)
		}
		unregisterWaiter(w)
		_ = writeBulkArray(w.Writer, []string{key, v})
	}
}

// ProcessBlockTimeouts replies null-array to expired waiters. Call from server cron.
func ProcessBlockTimeouts() {
	now := time.Now()
	seen := make(map[*BlockWaiter]struct{})
	var expired []*BlockWaiter
	for _, list := range blockWaiters {
		for _, w := range list {
			if _, ok := seen[w]; ok {
				continue
			}
			seen[w] = struct{}{}
			if w.Deadline.IsZero() {
				continue
			}
			if !now.Before(w.Deadline) {
				expired = append(expired, w)
			}
		}
	}
	for _, w := range expired {
		unregisterWaiter(w)
		_, _ = w.Writer.Write(RESP_NIL_ARRAY)
	}
}

// evalBlockingPop handles BLPOP / BRPOP.
// Returns blocked=true when the client was parked with no immediate reply.
// When allowBlock is false (e.g. inside EXEC), empty lists return a null array.
func evalBlockingPop(client *Client, args []string, w io.Writer, left bool, allowBlock bool) (blocked bool, err error) {
	cmdName := "blpop"
	if !left {
		cmdName = "brpop"
	}
	if len(args) < 2 {
		return false, fmt.Errorf("(error) ERR wrong number of arguments for '%s' command", cmdName)
	}
	if allowBlock && client == nil {
		return false, errors.New("(error) ERR blocking commands require a client")
	}

	timeoutStr := args[len(args)-1]
	keys := args[:len(args)-1]
	timeoutSec, err := strconv.ParseFloat(timeoutStr, 64)
	if err != nil {
		return false, errors.New("(error) ERR timeout is not a float or out of range")
	}
	if timeoutSec < 0 {
		return false, errors.New("(error) ERR timeout is negative")
	}

	side := listRight
	if left {
		side = listLeft
	}

	for _, key := range keys {
		obj := Get(key)
		if obj == nil {
			continue
		}
		if obj.Type != ObjList {
			return false, ErrWrongType
		}
		c, err := getListContainer(obj)
		if err != nil {
			return false, err
		}
		if c.Len() == 0 {
			continue
		}
		var (
			v  string
			ok bool
		)
		if side == listLeft {
			v, ok = c.PopLeft()
		} else {
			v, ok = c.PopRight()
		}
		if !ok {
			continue
		}
		if c.Len() == 0 {
			deleteKey(key)
		} else {
			bumpVersion(key)
		}
		return false, writeBulkArray(w, []string{key, v})
	}

	if !allowBlock {
		_, err := w.Write(RESP_NIL_ARRAY)
		return false, err
	}

	var deadline time.Time
	if timeoutSec > 0 {
		deadline = time.Now().Add(time.Duration(timeoutSec * float64(time.Second)))
	}

	waiter := &BlockWaiter{
		Client:   client,
		Writer:   w,
		Keys:     append([]string{}, keys...),
		Deadline: deadline,
		Left:     left,
	}
	registerWaiter(waiter)
	return true, nil
}

func resetBlockWaitersForTests() {
	blockWaiters = make(map[string][]*BlockWaiter)
}
