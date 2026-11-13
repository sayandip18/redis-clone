# mini-redis

A Redis clone written in Go, built as a learning project. It speaks the RESP protocol, so you can talk to it with the real `redis-cli`.

> Status: work in progress. See the roadmap below.

## Quick start

```bash
# Run the server (listens on :6379)
go run ./cmd/server

# In another terminal
redis-cli -p 6379 PING
# PONG
```

## Supported commands

| Category | Commands |
|---|---|
| Connection | `PING`, `ECHO` |
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
cmd/server/            # entry point
internal/resp/         # RESP parser and writer
internal/server/       # TCP server
internal/store/        # in-memory data store
internal/commands/     # command handlers
internal/persistence/  # AOF and RDB
```

## Roadmap

- [ ] TCP server + RESP protocol
- [ ] Strings (`SET`, `GET`, `DEL`, `INCR`, ...)
- [ ] Key expiration
- [ ] Lists, hashes, sets, sorted sets
- [ ] Persistence (AOF, RDB)
- [ ] Pub/sub, transactions, eviction
- [ ] Benchmarks and profiling

## License

MIT