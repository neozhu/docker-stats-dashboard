# Project Context

## Mission & Goals
- Deliver a lightweight way to observe container CPU and memory usage across many Docker hosts with <1s perceived latency.
- Keep setup friction near zero: a single agent binary per host, a static dashboard, and optional docker-compose for demos.
- Embrace a decentralized model; browsers connect straight to agents, so there is no hub, queue, or database to operate.

## Success Metrics
- First meaningful charts appear in under 10s after the dashboard loads and connects to at least one agent.
- One agent instance comfortably streams stats for 100 running containers with <10% agent CPU overhead.
- Operators can add or remove an agent endpoint from the dashboard UI without reloading the page.

## Non-Goals
- Historic trend storage, alerting, or long-term retention.
- Authentication, RBAC, or multi-tenant access controls in v1.
- Support for container runtimes other than Docker/Moby.

# Product Overview

## Primary Personas
- Platform/Infra engineers who manage several Docker hosts and need fast visibility.
- Developers reproducing production issues locally and wanting a quick glance at resource usage.
- SREs performing ad-hoc triage without installing heavy observability stacks.

## Core Scenarios
- Real-time comparison of resource usage across hosts during incident response.
- Spotting noisy neighbours or runaway containers on lab clusters.
- Temporary monitoring for short-lived environments (e.g., CI runners or staging).

## User Experience Highlights
- Dashboard lists agents as cards; containers under each card sorted by CPU by default.
- Users paste WebSocket URLs, which are persisted to `localStorage` for the next session.
- UI surfaces connection state (connecting, live, disconnected) along with last update timestamps.

## Data Flow Overview
1. Each agent samples Docker stats via the Docker SDK (or `docker stats` fallback) every second.
2. Agents broadcast structured JSON messages over WebSocket to all connected clients.
3. Browsers merge streams from multiple agents, normalize time, and render charts in real time.

# Architecture

## Runtime Topology
```
                     ws://host-a:8080
  +-----------+ <-------------------------+-----------+
  |           |                           |  Agent A  |
  |  Browser  |   ws://host-b:8080        +-----------+
  | Dashboard | <-------------------------+ Docker SDK|
  |           |                           |  (Host B) |
  +-----------+                           +-----------+
         ^                                     ^
         |                                     |
         +------------- ws://host-c:8080 ------+
```
- Browsers maintain one WebSocket per agent; agents do not coordinate with each other.
- Docker Engine access is through `/var/run/docker.sock` (bind-mounted when running inside Docker).

## Component Breakdown

**Go Agent**
- Targets Go 1.23+. Packaged as a single binary or Docker image.
- Collects stats via `github.com/docker/docker` client with 1s cadence.
- Broadcasts `container_stats_batch` payloads to all live WebSocket clients; includes lightweight heartbeats.
- Handles concurrent clients via goroutines; goal is zero shared mutable state beyond the stats cache.

**Web Dashboard (SvelteKit)**
- SvelteKit 2 + TypeScript, shadcn-svelte for primitives, Tailwind for styling.
- Uses writable stores for agent registry and incoming stats buffers; components subscribe reactively.
- Performs client-side smoothing/throttling to avoid over-rendering when many containers are active.
- Persists agent endpoints to `localStorage`; optionally backs up to a downloadable JSON for portability.

**Optional Reverse Proxy**
- Nginx or Traefik can terminate TLS and proxy WebSocket traffic for internet-facing setups.
- Proxy configuration is out of scope for v1 but documented for operators.

## Data Contracts

### `container_stats_batch`
```
{
  "type": "container_stats_batch",
  "agent_id": "host-a",
  "sent_at": "2025-10-15T10:00:00Z",
  "containers": [
    {
      "id": "abc123",
      "name": "nginx",
      "cpu_pct": 24.5,
      "mem_bytes": 128345600,
      "mem_limit_bytes": 536870912,
      "uptime_secs": 532
    }
  ],
  "agent_metrics": {
    "cpu_pct": 4.2,
    "mem_bytes": 66469888
  }
}
```
- Every message includes a `type` field for extensibility.
- `agent_id` defaults to host name but can be overridden via flag/env.
- Empty `containers` arrays are valid when the host is idle; UI shows a placeholder.

### `agent_status`
```
{
  "type": "agent_status",
  "agent_id": "host-a",
  "sent_at": "2025-10-15T10:00:05Z",
  "uptime_secs": 3600,
  "version": "v1.0.0",
  "features": ["container_stats"]
}
```
- Broadcast every 30 s; helps dashboards surface basic health without stats data.
- Future message types should be described here and kept backwards compatible.

# Operational Constraints

## Security & Privacy
- Agents assume a trusted LAN; no auth or TLS termination by default. Document reverse-proxy hardening for any exposed deployment.
- Agents never shell out or accept commands; all payloads are read-only stats.
- No persistent secrets are stored. The only browser storage is the list of agent URLs.

## Performance Targets
- Sampling interval fixed at 1s in v1; make it configurable only after validating load impact.
- Memory footprint target for the agent is <50MB RSS while tracking 100 containers.
- Dashboard should remain responsive (<16ms render budget) with three agents x 100 containers.

## Failure Handling
- Agent restarts should not require dashboard refresh; the client auto-retries every 5s with exponential backoff.
- Dashboard surfaces partial failures (e.g., one agent offline) without blocking other connections.
- If Docker stats streaming fails, agent falls back to polling individual containers before exiting.

# Development Workflow

## Planned Repository Layout
```
/agent/            # Go source, Dockerfile, Makefile
/web/              # SvelteKit app, component library, tests
/deploy/           # docker-compose.yml, nginx sample configs
/openspec/         # Specs, proposals, tasks (managed by OpenSpec)
/scripts/          # Tooling helpers for lint/test/build
```
- Keep shared schemas (TypeScript + Go) in `/agent/internal/schema` and `/web/src/lib/schema` generated from a common JSON schema once we introduce codegen.

## Toolchain & Requirements
- Go 1.23+, Node 22+, Docker Engine 24+, docker-compose v2.
- Recommended helpers: `air` for hot reload of the agent, `pnpm` for frontend installs (falls back to npm if unavailable).
- Install formatting/linting hooks via `pre-commit` (optional but encouraged).

## Local Setup
1. `make bootstrap` (installs Go tools, runs `pnpm install` in `/web`).
2. `make dev` launches the agent with live reload and the dashboard on `http://localhost:5173`.
3. Sample docker-compose (`deploy/docker-compose.yml`) starts one agent and the dashboard for manual testing.
- Windows users run the agent with `--docker-endpoint=npipe:////./pipe/docker_engine`.

## Testing Strategy
- **Agent**: Go `testing` package, table-driven tests for stat parsing and broadcaster behavior; use `gotest.tools/v3` assertions.
- **Frontend**: Vitest for stores/utilities, Playwright for smoke E2E (connect to mocked agent server).
- **Integration**: Deterministic `docker-compose.test.yml` spins up a mock workload plus the dashboard; assertions via Playwright or curl scripts.
- **Performance**: Manual stress script (`scripts/loadtest_agent.go`) launches synthetic containers to validate 1s cadence.

## Coding Conventions
- Run `gofmt`, `goimports`, and `golangci-lint run ./...` before committing Go code.
- Frontend uses Prettier (2 spaces, single quotes) and ESLint with Svelte and TypeScript plugins.
- Shared rule: every WebSocket payload must include `type`, `sent_at`, and a monotonically increasing `sequence` once implemented.
- Commit messages follow Conventional Commits; include scope `agent`, `web`, `deploy`, or `docs`.
- Avoid introducing new third-party dependencies without noting rationale in PR description and updating this document if the dependency is foundational.

# Release & Deployment
- Version artifacts with semver; tag releases as `vX.Y.Z`. Agents embed version via `-ldflags "-X main.version=$(TAG)"`.
- Docker images published to GHCR: `ghcr.io/<org>/docker-monitor-agent` and `.../dashboard`.
- Release pipeline steps:
  1. Run `make ci` (lint + test) for both subprojects.
  2. Build multi-arch images (`linux/amd64`, `linux/arm64`) via `docker buildx bake release`.
  3. Push static dashboard bundle to GitHub Pages or S3 (documented in `/deploy/README.md`).
- Patch releases must include changelog entries summarizing user-facing behavior changes.

# Observability & Troubleshooting
- Agent exposes a `/healthz` HTTP endpoint (localhost-only) returning build info and last Docker poll timestamp.
- Dashboard surfaces in-app debug panel (toggle via `?debug=true`) showing raw messages and WebSocket state.
- For verbose diagnostics, run the agent with `--log-level=debug` to print each payload header and dispatch time.

# Known Gaps & Future Work
- Consider optional TLS termination + token auth for internet-facing agents.
- Evaluate including disk I/O statistics once the Docker SDK support is validated.
- Research WebRTC data channels as a potential transport to pass through restrictive networks.
- Add localization support to the dashboard after v1.
- Explore exporting summarized metrics to Prometheus as an optional module.

# Change Control
- Any architectural change (e.g., adding a hub service or persistence) requires a new OpenSpec change proposal.
- Minor bug fixes and styling tweaks can proceed without proposals but must still reference relevant specs once they exist.
- Keep this document aligned with reality; update sections whenever tooling, workflows, or constraints shift.
