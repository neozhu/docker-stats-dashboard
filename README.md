# Docker Stats Dashboard

A lightweight, two-piece monitoring stack for real-time Docker container metrics. The Go agent streams stats straight from each host, while the SvelteKit dashboard fans them into a single Server-Sent Events (SSE) feed for the browser. Deployment stays simple—no external queues, databases, or heavyweight dependencies required.

> **Note:** Full coding for this project was completed entirely by **AI CODEX** under **OpenSpec** guidance.

![Dashboard preview placeholder](docs/images/dashboard-preview.png)

*Screenshot placeholder — update with a fresh capture when available.*

## Table of contents

1. [Why this project?](#why-this-project)
2. [Architecture at a glance](#architecture-at-a-glance)
3. [Repository layout](#repository-layout)
4. [Quick start](#quick-start)
5. [Configuration reference](#configuration-reference)
6. [Development scripts](#development-scripts)
7. [Continuous integration](#continuous-integration)
8. [Production notes](#production-notes)
9. [Roadmap ideas](#roadmap-ideas)

## Why this project?

- **Minimal footprint.** The agent is a stateless Go binary that runs alongside workloads without resource contention. The dashboard is a SvelteKit app that streams metrics over SSE; you deploy a single lean Node process per environment.
- **Simple operations.** Discover hosts, track connection state, and surface per-container CPU/memory stats without provisioning extra infrastructure.
- **Frictionless deployment.** Run the agent as a Docker container or binary, point the dashboard at your agent URLs via one environment variable, and you are operational in minutes.

## Architecture at a glance

```
Docker Host(s)             Dashboard Host
--------------             ---------------
[docker-agent] --WS-->  ┐  SvelteKit server --SSE--> Browser UI
[docker-agent] --WS-->  ┘
```

1. Each Docker host runs `docker-agent`, a Go service that polls the Docker Engine and publishes structured metrics over WebSocket.
2. The dashboard's Node runtime maintains outbound WebSocket connections to every configured agent.
3. Metrics and status updates are multiplexed into a single SSE stream (`/stream/agents`), which the browser consumes.

## Repository layout

| Path        | Description |
| ----------- | ----------- |
| `agent/`    | Go WebSocket agent that scrapes the Docker Engine API and emits stats batches plus heartbeat updates. |
| `web/`      | SvelteKit dashboard with an in-process hub that fans agent traffic into an SSE endpoint. |
| `openspec/` | OpenSpec change proposals and specs that guided AI-assisted development. |

## Quick start

### 1. Run an agent on a Docker host

```bash
cd agent
make run     # or: go run .
```

Default listen address: `ws://localhost:8080/ws`. Provide a friendly label with `AGENT_HOST_LABEL="host-a"` if desired.

To containerize instead:

```bash
cd agent
docker build -t docker-agent:dev .
docker run --rm \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  docker-agent:dev
```

### 2. Configure and start the dashboard

```bash
cd web
cp .env.local.example .env.local
# Edit AGENT_ENDPOINTS, e.g.
# AGENT_ENDPOINTS=ws://host-a:8080/ws;ws://host-b:8080/ws
pnpm install
pnpm dev
```

Open `http://localhost:5173` to view live metrics. The development server fans in all agents and exposes `/stream/agents` for the UI.

### 3. Optional: docker-compose for local demo

```yaml
docker-compose:
  services:
    agent:
      build: ./agent
      environment:
        AGENT_HOST_LABEL: lab-node-1
      volumes:
        - /var/run/docker.sock:/var/run/docker.sock
    dashboard:
      build: ./web
      environment:
        AGENT_ENDPOINTS: ws://agent:8080/ws
      ports:
        - "5173:5173"
```

> The dashboard only needs network access to each agent URL; no shared state is required.

## Configuration reference

### Agent (`agent/`)

| Flag | Env Var | Default | Purpose |
| ---- | ------- | ------- | ------- |
| `--docker-endpoint` | `AGENT_DOCKER_ENDPOINT` | `unix:///var/run/docker.sock` | Docker Engine API endpoint. |
| `--listen` | `AGENT_LISTEN_ADDR` | `:8080` | HTTP/WebSocket listen address. |
| `--host-label` | `AGENT_HOST_LABEL` | host hostname | Human-friendly name used in the UI. |
| `--poll-interval` | `AGENT_POLL_INTERVAL` | `500ms` | Sampling cadence for container stats. |
| `--max-workers` | `AGENT_MAX_WORKERS` | `16` | Concurrent stats workers. |

### Dashboard (`web/`)

| Variable | Example | Notes |
| -------- | ------- | ----- |
| `AGENT_ENDPOINTS` | `agent-a|DB Node|ws://10.0.0.15:8080/ws;agent-b|Cache|ws://10.0.0.16:8080/ws` | Semicolon-separated list. Omit `id|label|` to auto-generate defaults. |

## Development scripts

| Location | Command | Purpose |
| -------- | ------- | ------- |
| `agent/` | `make test` | Run Go unit tests. |
| `agent/` | `make build` | Produce the `docker-agent` binary. |
| `web/`   | `pnpm lint` | Format and lint SvelteKit code. |
| `web/`   | `pnpm check` | Run Svelte type checks. |
| `web/`   | `pnpm test` | Execute Vitest unit tests. |

## Continuous integration

| Workflow | Scope | What it covers |
| -------- | ----- | -------------- |
| `Agent CI` | `agent/` | Runs `go mod verify` and `go build ./...` on pushes and pull requests touching agent code. |
| `Web CI` | `web/` | Installs dependencies via pnpm, then runs linting and type checks for the SvelteKit app. |

## Production notes

- Prefer `@sveltejs/adapter-node` when deploying the dashboard so the SSE endpoint runs in a long-lived Node process.
- Reverse proxies (Nginx, Caddy, Traefik) can forward `/stream/agents` like any HTTP request; keep `Content-Type: text/event-stream` intact.
- Horizontal scale is as simple as running multiple dashboard instances pointing at the same agents. If you need shared history, plug a message bus (Redis, NATS, etc.) behind the hub later.

## Roadmap ideas

- In-memory history buffers for sparkline charts.
- Token-based auth on the SSE endpoint.
- Optional WebSocket frontend for command/control (pause or filter streams).

---

This repository—and every line of code within—was implemented by AI CODEX following the OpenSpec workflow. Contributions are welcome via pull requests.
