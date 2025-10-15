## 1. Scaffold Agent Project
- [ ] 1.1 Initialise Go module under `/agent` with Go 1.23, define `main.go`, `go.mod`, and shared config structure.
- [ ] 1.2 Add CLI flag/env parsing for Docker endpoint, listen address, host label, poll interval, and log level.
- [ ] 1.3 Implement structured logging setup and graceful shutdown handling (signals, context cancellation).

## 2. Collect Docker Stats
- [ ] 2.1 Use Docker SDK to enumerate running containers and open streaming stats (`ContainerStats`) with timeout handling.
- [ ] 2.2 Normalise CPU/memory usage into the `container_stats_batch` schema (include sequence, uptime, limits).
- [ ] 2.3 Maintain lightweight cache of last stats to satisfy outgoing requests even if a poll fails temporarily.

## 3. WebSocket Broadcast Layer
- [ ] 3.1 Expose WebSocket endpoint (e.g., `/ws`) to broadcast JSON batches to multiple clients concurrently.
- [ ] 3.2 Implement heartbeat / agent status messages every 30s and detect disconnected clients.
- [ ] 3.3 Enforce 1s broadcast cadence with backpressure handling so slow clients do not block others.

## 4. Packaging & Quality
- [ ] 4.1 Write unit tests for stats conversion and message encoding; add integration smoke test hitting a fake Docker API.
- [ ] 4.2 Provide Dockerfile, docker-compose example, and Makefile targets (`make build`, `make run`) under `/agent`.
- [ ] 4.3 Document setup, permissions, and operational notes in `/agent/README.md`; update root README if necessary.
