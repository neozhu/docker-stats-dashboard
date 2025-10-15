# Docker Performance Monitor Agent

This Go service streams Docker container CPU and memory statistics to dashboard clients over WebSocket. It is designed to be lightweight, stateless, and safe to run alongside workloads on any Docker host.

## Prerequisites

- Go 1.25+ (local development)
- Access to the Docker Engine socket (`/var/run/docker.sock`) or TCP API
- `make` (optional but recommended)

> The repository CI currently runs on Go 1.18; when working locally use Go 1.25 or newer to match production targets.

## Getting Started

Install dependencies and run the agent locally:

```bash
cd agent
go mod tidy     # first time only
go run .
```

The agent listens on `http://localhost:8080/ws` by default and emits mock-friendly JSON that matches the dashboard schema.

### Using make

```bash
make build              # builds bin/docker-agent
make run                # go run .
make test               # go test ./...
```

## Configuration

Flags accept environment variable equivalents (`AGENT_*`). Defaults are shown below.

| Flag               | Env Var                | Default                         | Description                                    |
| ------------------ | ---------------------- | ------------------------------- | ---------------------------------------------- |
| `--docker-endpoint`| `AGENT_DOCKER_ENDPOINT`| `unix:///var/run/docker.sock`   | Docker Engine endpoint                         |
| `--listen`         | `AGENT_LISTEN_ADDR`    | `:8080`                         | HTTP/WebSocket listen address                  |
| `--host-label`     | `AGENT_HOST_LABEL`     | local hostname                  | Friendly label advertised to dashboards        |
| `--poll-interval`  | `AGENT_POLL_INTERVAL`  | `1s`                            | Sampling cadence for container stats           |
| `--log-level`      | `AGENT_LOG_LEVEL`      | `info`                          | Log level (`debug`, `info`, `warn`, `error`)   |

Example:

```bash
docker-agent \
  --docker-endpoint tcp://192.168.1.10:2375 \
  --poll-interval 2s \
  --host-label staging-a
```

## Running with Docker

```bash
docker build -t docker-agent:dev .
docker run --rm \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  docker-agent:dev
```

### docker-compose sample

```yaml
services:
  agent:
    build: .
    ports:
      - "8080:8080"
    environment:
      AGENT_HOST_LABEL: "lab-node-1"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
```

## Observability

- `/healthz` returns basic health data for liveness checks.
- Heartbeat messages (`agent_status`) publish version, uptime, and feature list every 30 seconds.
- Structured JSON logs are emitted to stdout.

## Testing

```bash
make test
```

Unit tests cover configuration parsing and stat conversion logic. Integration tests that exercise Docker APIs can be added later once a fake daemon is available.
