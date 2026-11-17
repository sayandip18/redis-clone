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
| Connection | `PING`, `INFO`, `SHUTDOWN` |
| Strings | `SET`, `GET`, `INCR`, `APPEND`, `STRLEN`, `GETRANGE`, `SETRANGE` |
| Keys | `DEL`, `EXPIRE`, `TTL`, `OBJECT` |
| Lists | `LPUSH`, `RPUSH`, `LPOP`, `RPOP`, `LLEN`, `LRANGE`, `LINDEX`, `LSET`, `LINSERT`, `LTRIM`, `LREM`, `BLPOP`, `BRPOP` |
| Sets | `SADD`, `SREM`, `SISMEMBER`, `SCARD`, `SMEMBERS`, `SRANDMEMBER`, `SPOP`, `SINTER`, `SUNION`, `SDIFF`, `SINTERSTORE`, `SUNIONSTORE`, `SDIFFSTORE` |
| Transactions | `MULTI`, `EXEC`, `DISCARD`, `WATCH` |
| Persistence | `BGREWRITEAOF` |

String encodings: `int` / `embstr` / `raw` (SDS). List encodings: `ziplist` → `quicklist`. Set encodings: `intset` → `hashtable`.

## Development

```bash
go build ./...
go test ./... -race
go vet ./...
```

## Project layout

```
main.go              # entry point
config/              # host/port and encoding thresholds
server/              # TCP servers (sync + async/epoll)
internal/resp/       # RESP decode/encode
internal/core/       # store, encodings, command handlers
```

## Roadmap

- [x] TCP server + RESP protocol
- [x] `PING`
- [x] Strings (`SET`, `GET`, `DEL`, `INCR`, ...)
- [x] Key expiration
- [x] Lists (ziplist / quicklist) and sets (intset / hashtable)
- [ ] Hashes, sorted sets
- [x] Persistence (AOF rewrite)
- [ ] RDB
- [x] Transactions, eviction
- [ ] Pub/sub
- [ ] Benchmarks and profiling

## License

MIT
