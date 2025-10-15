## Why
- We need a concrete starting point for the browser dashboard described in the project brief so future UI features can build on a consistent SvelteKit baseline.
- Using the official SvelteKit CLI (`npx sv create`) ensures the scaffold follows the recommended project structure and tooling versions.
- Establishing Tailwind, shadcn-svelte components, and shared typing conventions early avoids churn when implementing live Docker stats views.

## What Changes
- Create a new SvelteKit 2 project under `/web` using `npx sv create` with TypeScript, ESLint, Playwright, and Vite test tooling enabled.
- Integrate TailwindCSS and shadcn-svelte starter configuration so shared UI primitives are ready for later dashboards.
- Add placeholder routes, layouts, and stores that reflect the planned dashboard shell (agent list page, connection status banner).
- Document development commands (`pnpm dev`, `pnpm test`) and bootstrap steps in `/web/README.md`.

## Impact
- New frontend dependencies (Node-based) and configuration files will appear under `/web`.
- Developers will install Node 22+ and pnpm to work on the dashboard; CI will need to run frontend lint/test jobs.
- No backend/agent behavior changes; Docker hosts remain unaffected until UI functionality is implemented.
