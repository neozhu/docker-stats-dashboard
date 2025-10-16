## Why
- Removing an agent from the dashboard currently triggers the same disconnect path as an unexpected network failure, which schedules a reconnect attempt and briefly renders the card in an error state even though the user opted to delete it.
- The lingering reconnect timer keeps the component logic and websocket transport alive for several seconds, delaying removal of in-memory snapshots and generating noisy console warnings if the endpoint is unreachable.
- We need to ensure a manual removal tears down the agent cleanly so the UI reflects the user's intent immediately and no background work continues for that endpoint.

## What Changes
- Update the Web Dashboard `handleRemoveAgent` flow to mark user-initiated removals before closing sockets so the disconnect handler skips reconnect scheduling.
- Extend the agent registry utilities so a removed agent's state (status, latest snapshots, sequence counters, timers) is cleared synchronously and the entry is deleted from `localStorage`.
- Add regression coverage (store-level unit test or component test) that confirms removing an agent does not attempt reconnection and that the card disappears without flashing an error state.

## Impact
- Only the SvelteKit dashboard is affected; the Go agent and transport protocol remain unchanged.
- Frontend logic will gain a small amount of additional bookkeeping to differentiate manual removals from transient disconnects.
- No new dependencies are expected; the change should remain within existing SvelteKit and Vitest tooling.
