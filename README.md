# redis-clone

A Redis clone written in Go, built as a learning project. It speaks the RESP protocol, so you can talk to it with the real `redis-cli`.

> Status: work in progress. See the roadmap below.

## Quick start

```bash
# Run the server (listens on :7379 by default)
go run .

# In another terminal
redis-cli -p 7379 PING
# PONG
```

Optional flags:

```bash
go run . -host 127.0.0.1 -port 7379
```

## Supported commands

| Category | Commands |
|---|---|
| Connection | `PING` |
| Strings | _coming soon_ |

This table is updated as features are added.

## Development

```bash
go build ./...
go test ./... -race
go vet ./...
```

## Project layout

```
main.go              # entry point
config/              # host/port defaults
server/              # synchronous TCP server
internal/resp/       # RESP decode/encode
internal/core/       # command types and handlers
```

## Roadmap

- [x] TCP server + RESP protocol
- [x] `PING`
- [ ] Strings (`SET`, `GET`, `DEL`, `INCR`, ...)
- [ ] Key expiration
- [ ] Lists, hashes, sets, sorted sets
- [ ] Persistence (AOF, RDB)
- [ ] Pub/sub, transactions, eviction
- [ ] Benchmarks and profiling

## License

MIT
