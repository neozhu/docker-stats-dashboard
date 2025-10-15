## ADDED Requirements

### Requirement: Periodic Container Sampling
The Go agent MUST sample CPU and memory stats for each running container on its Docker host at a 1s cadence by default.

#### Scenario: Successful polling
- **GIVEN** the agent is running with access to the Docker socket
- **WHEN** at least one container is running
- **THEN** every second the agent queries Docker for fresh stats and stores the latest values for broadcast.

#### Scenario: Poll failure recovery
- **GIVEN** a temporary Docker API failure occurs during sampling
- **WHEN** the agent retries after the failure
- **THEN** the agent logs the error at warn level, keeps the previous sample, and resumes polling on the next interval.

### Requirement: WebSocket Broadcasting
The agent MUST expose a WebSocket endpoint that streams `container_stats_batch` JSON payloads and periodic `agent_status` messages to all connected clients.

#### Scenario: Broadcast cadence
- **GIVEN** a client subscribes to the `/ws` endpoint
- **WHEN** the agent has fresh stats
- **THEN** the client receives a `container_stats_batch` message at most 1s after the stats were sampled.

#### Scenario: Multiple clients
- **GIVEN** three dashboard sessions connect concurrently
- **WHEN** any client becomes slow or disconnects
- **THEN** other clients continue receiving messages without delay, and the agent cleans up the slow connection.

#### Scenario: Agent status heartbeat
- **GIVEN** the agent has no new container stats to send
- **WHEN** 30s elapse since the last heartbeat
- **THEN** it emits an `agent_status` message with host label, uptime, version, and feature list.

### Requirement: Configurable Runtime
The agent MUST support configuration via flags and environment variables for Docker endpoint, listen address, host label, poll interval, and log level.

#### Scenario: CLI overrides
- **GIVEN** the agent is started with `--docker-endpoint tcp://host:2375 --poll-interval=2s`
- **WHEN** it runs
- **THEN** it connects to the specified Docker endpoint and samples every 2 seconds.

#### Scenario: Environment defaults
- **GIVEN** no flags are provided but `AGENT_HOST_LABEL=staging-a`
- **WHEN** the agent runs
- **THEN** outgoing messages include `agent_label: "staging-a"`.

### Requirement: Packaging & Docs
The repository MUST include build tooling and documentation to run the agent locally or via Docker.

#### Scenario: Local build
- **GIVEN** a developer with Go 1.23 executes `make build` in `/agent`
- **WHEN** the command succeeds
- **THEN** a binary (e.g., `bin/docker-agent`) is produced.

#### Scenario: Docker image available
- **GIVEN** a user runs `docker build` using the provided Dockerfile
- **WHEN** they start the image with the Docker socket mounted
- **THEN** the container runs the agent and exposes the WebSocket endpoint as documented.

#### Scenario: README guidance
- **GIVEN** someone opens `/agent/README.md`
- **WHEN** they follow the setup instructions
- **THEN** they learn the required permissions, configuration flags, and how to launch the agent in Docker or systemd.
