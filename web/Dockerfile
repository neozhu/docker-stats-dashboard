# syntax=docker/dockerfile:1

FROM node:20-alpine

WORKDIR /app

# Install pnpm via corepack and required compatibility libs
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"

RUN apk add --no-cache libc6-compat \
    && corepack enable

# Copy lockfile & manifest first for better caching
COPY package.json pnpm-lock.yaml ./

# IMPORTANT: Do NOT set NODE_ENV=production before this install.
# We need devDependencies (vite, svelte, etc.) to run the build & preview.
# Setting NODE_ENV=production would cause pnpm to skip dev deps and the build would fail.
RUN pnpm install --frozen-lockfile

COPY svelte.config.js tsconfig.json vite.config.ts eslint.config.js ./
COPY components.json ./components.json
COPY static ./static
COPY src ./src

RUN pnpm build

# Expose the preview server port
EXPOSE 4173

# Optionally set NODE_ENV at runtime (after dependencies installed) if desired
ENV NODE_ENV=production

CMD ["pnpm", "preview", "--host", "0.0.0.0", "--port", "4173"]
