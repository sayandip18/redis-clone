package core

// Client holds per-connection state for transactions and watches.
type Client struct {
	InTx     bool
	Queue    []*RedisCmd
	Watching map[string]uint64 // key → version at WATCH time
	Aborted  bool              // queue-time error; EXEC must fail
}

// NewClient returns a client with empty transaction state.
func NewClient() *Client {
	return &Client{}
}

func (c *Client) clearTx() {
	if c == nil {
		return
	}
	c.InTx = false
	c.Queue = nil
	c.Aborted = false
}

func (c *Client) clearWatches() {
	if c == nil {
		return
	}
	c.Watching = nil
}

func (c *Client) clearTxAndWatches() {
	c.clearTx()
	c.clearWatches()
}

// watchesIntact reports whether every watched key still has its snapshotted version.
func (c *Client) watchesIntact() bool {
	if c == nil || len(c.Watching) == 0 {
		return true
	}
	for key, ver := range c.Watching {
		if KeyVersion(key) != ver {
			return false
		}
	}
	return true
}
