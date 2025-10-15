## ADDED Requirements

### Requirement: Dashboard Shell
The project MUST include a SvelteKit application that renders a shared layout for the Docker Performance Monitor dashboard.

#### Scenario: SvelteKit project scaffolded
- **GIVEN** the repository is freshly cloned
- **WHEN** a developer runs `pnpm install` and `pnpm dev` inside `/web`
- **THEN** the app starts with a root layout that renders a header, main content area, and global styles consistent with Tailwind configuration.

#### Scenario: CLI origin documented
- **GIVEN** the SvelteKit app originated from `npx sv create`
- **WHEN** a contributor inspects `/web/README.md`
- **THEN** the README references the CLI scaffold origin and outlines how to regenerate or upgrade the scaffold.

### Requirement: Agent Registry Placeholder
The dashboard MUST provide placeholder UI and state management for listing agent WebSocket endpoints and connection state.

#### Scenario: Agent list placeholder visible
- **GIVEN** no agents have been configured yet
- **WHEN** the dashboard loads
- **THEN** a placeholder agent list component is visible, describing how to add endpoints and showing an empty-state message.

#### Scenario: Local store for agent endpoints
- **GIVEN** a user adds an agent endpoint via the placeholder UI
- **WHEN** the app persists state
- **THEN** the endpoint value is stored in a Svelte store that reads from and writes to `localStorage`, ready for future live connections.
