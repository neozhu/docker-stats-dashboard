## Why
- The dashboard relies on live container metrics; we need an agent that streams CPU and memory stats from Docker hosts.
- Implementing the agent now unblocks frontend integration and validates the JSON contracts defined in the project brief.
- Using the Docker SDK with efficient streaming keeps host overhead low while meeting the 1s cadence target.

## What Changes
- Build a Go service under `/agent` that connects to the local Docker Engine and streams container stats every second.
- Define WebSocket broadcasting so multiple clients receive `container_stats_batch` and `agent_status` messages.
- Add configuration for Docker socket path, poll interval, and host label via flags and environment variables.
- Provide Dockerfile, Makefile targets, and sample systemd/docker-compose definitions for running the agent.

## Impact
- New Go module and dependencies (Docker SDK, gorilla/websocket) increase build tooling surface.
- Hosts running the agent need read access to `/var/run/docker.sock` (or remote API); document security considerations.
- Frontend clients gain real-time data; ensure message format stays aligned with existing TypeScript typings.
