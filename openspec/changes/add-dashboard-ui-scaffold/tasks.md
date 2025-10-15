## 1. Scaffold SvelteKit Project
- [ ] 1.1 Run `npx sv create web` (or equivalent) to generate a SvelteKit 2 project with TypeScript, ESLint, Prettier, Vitest, and Playwright enabled.
- [ ] 1.2 Configure the new app to use `pnpm` (preferred) with Node 22+, and ensure lockfile and package scripts align with repo conventions.
- [ ] 1.3 Add a `/web/.tool-versions` or `.nvmrc` if needed so contributors pick up the right runtime versions.

## 2. Establish UI Foundation
- [ ] 2.1 Install and configure TailwindCSS, including base styles, PostCSS config, and Svelte-specific setup.
- [ ] 2.2 Integrate shadcn-svelte (or an equivalent component set) and add an example component to confirm the pipeline works.
- [ ] 2.3 Define global layout (`src/routes/+layout.svelte`) with shared shell elements (header, theme classes, safe area padding).
- [ ] 2.4 Create placeholder routes/components for the dashboard landing page, agent cards, and connection status banner.

## 3. Set Up State & Utilities
- [ ] 3.1 Implement initial Svelte stores for agent endpoints and connection state (in-memory + localStorage hydration stubs).
- [ ] 3.2 Add shared TypeScript types mirroring the `container_stats_batch` message shape so future features can rely on them.
- [ ] 3.3 Provide a lightweight mock WebSocket service or utility to support local development without a running agent.

## 4. Tooling & Documentation
- [ ] 4.1 Ensure linting (`pnpm lint`), unit tests (`pnpm test`), and Playwright smoke tests (`pnpm test:e2e` placeholder) run successfully.
- [ ] 4.2 Document install, dev, lint, and test commands in `/web/README.md`, including prerequisites (Node, pnpm).
- [ ] 4.3 Add the new frontend jobs or commands to the repo-level CI script or document follow-up work if CI is out of scope.
