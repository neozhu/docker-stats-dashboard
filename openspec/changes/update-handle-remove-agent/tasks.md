## 1. Confirm Existing Bug
- [x] 1.1 Reproduce the current remove-agent workflow and capture how `handleRemoveAgent` triggers a reconnect/error state.
- [x] 1.2 Identify the minimal state needed to flag manual removals before the websocket disconnect handler runs.

## 2. Update Dashboard Logic
- [x] 2.1 Adjust `handleRemoveAgent` (and related socket helpers) to mark an agent as intentionally removed prior to closing the socket.
- [x] 2.2 Skip reconnect scheduling and status updates for agents flagged as removed while ensuring the entry is pruned from `agentRegistry`.
- [x] 2.3 Clear cached snapshots, sequence counters, and timers immediately so no stale stats remain after removal.

## 3. Regression Coverage & Cleanup
- [x] 3.1 Add a unit or component test that removes an agent and asserts no reconnect attempts or error flashes occur.
- [x] 3.2 Verify localStorage persistence reflects the removal and document the behavior in developer notes if clarification is useful.
- [ ] 3.3 Run dashboard lint/test commands to confirm the changes integrate cleanly.
