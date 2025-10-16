## ADDED Requirements

### Requirement: Manual Agent Removal Cleanup
The Web Dashboard MUST treat user-triggered agent removals as final and clear all related connection state immediately.

#### Scenario: Remove from dashboard
- **GIVEN** an agent card is visible and currently connected
- **WHEN** the operator clicks the "Remove" action for that agent
- **THEN** the card disappears from the list within one second
- **AND** no reconnect attempt is scheduled and the status badge never flips to `error` as part of the removal.

#### Scenario: Pending reconnect suppressed
- **GIVEN** a reconnect timer is pending because the agent recently disconnected
- **WHEN** the operator removes that agent before the timer fires
- **THEN** the timer is cancelled and no websocket connection is re-established for that agent id.

#### Scenario: Persistence updated
- **GIVEN** the agent endpoint exists in persisted storage
- **WHEN** it is removed via the dashboard action
- **THEN** the agent entry is deleted from `localStorage` and does not reappear on a subsequent page reload.
