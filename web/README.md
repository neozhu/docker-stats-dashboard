# Docker Performance Monitor – Web Dashboard

This SvelteKit app was scaffolded with `npx sv create web` and extended with TailwindCSS and shadcn-svelte components. It provides the browser dashboard shell for the Docker Performance Monitor project.

## Prerequisites

- Node.js 22 (see `.nvmrc`)
- `pnpm` (Corepack users can run `corepack enable pnpm`)

## Install & Run

```bash
pnpm install
pnpm dev
```

The development server runs on `http://localhost:5173`. A mock WebSocket feed powers the dashboard until the Go agent integration is completed.

## Available Scripts

| Command            | Description                                       |
| ------------------ | ------------------------------------------------- |
| `pnpm dev`         | Start the Vite dev server                         |
| `pnpm build`       | Generate a production build                       |
| `pnpm preview`     | Preview the production build                      |
| `pnpm lint`        | Run Prettier and ESLint                           |
| `pnpm check`       | Run Svelte type checks                            |
| `pnpm test`        | Execute Vitest unit tests                         |
| `pnpm test:e2e`    | Placeholder Playwright command (no specs yet)     |

## Component Library

UI primitives are generated via `pnpm dlx shadcn-svelte@latest init` and `pnpm dlx shadcn-svelte add ...`. Regenerate components or update the registry with the same commands.

## Re-scaffolding

If the project ever needs a fresh scaffold, run:

```bash
pnpx sv@latest create web --template minimal --types ts
```

Then re-apply the TailwindCSS add-on (`pnpm dlx sv add tailwindcss`) and re-run the shadcn initialisation step described above.

Everything you need to build a Svelte project, powered by [`sv`](https://github.com/sveltejs/cli).

## Creating a project

If you're seeing this, you've probably already done this step. Congrats!

```sh
# create a new project in the current directory
npx sv create

# create a new project in my-app
npx sv create my-app
```

## Developing

Once you've created a project and installed dependencies with `npm install` (or `pnpm install` or `yarn`), start a development server:

```sh
npm run dev

# or start the server and open the app in a new browser tab
npm run dev -- --open
```

## Building

To create a production version of your app:

```sh
npm run build
```

You can preview the production build with `npm run preview`.

> To deploy your app, you may need to install an [adapter](https://svelte.dev/docs/kit/adapters) for your target environment.
