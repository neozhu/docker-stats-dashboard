# syntax=docker/dockerfile:1

FROM node:20-alpine

WORKDIR /app

ENV NODE_ENV=production
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"

RUN apk add --no-cache libc6-compat \
    && corepack enable

COPY package.json pnpm-lock.yaml ./

RUN pnpm install --frozen-lockfile

COPY svelte.config.js tsconfig.json vite.config.ts eslint.config.js ./
COPY components.json ./components.json
COPY static ./static
COPY src ./src

RUN pnpm build

EXPOSE 4173

CMD ["pnpm", "preview", "--host", "0.0.0.0", "--port", "4173"]
