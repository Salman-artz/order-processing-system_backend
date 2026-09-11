# ────────────────────────────────────────────────────────────────────────────
#  Stage 1 — deps
#  Install production + dev dependencies with npm ci for reproducible builds.
#  We use the full node image here because some npm packages need native tools.
# ────────────────────────────────────────────────────────────────────────────
FROM node:20-alpine AS deps

# Check https://github.com/nodejs/docker-node/tree/b4117f9333da4138b03a546ec926ef50a31506c3#nodealpine
RUN apk add --no-cache libc6-compat

WORKDIR /app

# Copy lockfiles first to exploit Docker layer caching
COPY package.json package-lock.json ./
RUN npm ci

# ────────────────────────────────────────────────────────────────────────────
#  Stage 2 — builder
#  Build the Next.js production bundle.
#
#  IMPORTANT: NEXT_PUBLIC_* env vars are embedded into the JS bundle at
#  BUILD TIME (not runtime), so they must be passed here as build args and
#  then exported as env vars before `next build` runs.
#  See: https://nextjs.org/docs/app/building-your-application/configuring/environment-variables
# ────────────────────────────────────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

# Receive build-time variables from docker-compose (or `docker build --build-arg`)
ARG NEXT_PUBLIC_API_BASE_URL=http://host.docker.internal:8080/api/v1
ARG NEXT_PUBLIC_WS_URL=ws://host.docker.internal:8080/ws
ARG NEXT_PUBLIC_USE_MOCK=false

# Export them so Next.js compiler picks them up
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_WS_URL=$NEXT_PUBLIC_WS_URL
ENV NEXT_PUBLIC_USE_MOCK=$NEXT_PUBLIC_USE_MOCK

# Next.js telemetry — disabled in CI/production to avoid noise
ENV NEXT_TELEMETRY_DISABLED=1

# Copy installed node_modules from deps stage (avoids re-downloading)
COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN npm run build

# ────────────────────────────────────────────────────────────────────────────
#  Stage 3 — runner
#  Minimal runtime image. Only contains what is strictly needed to serve the
#  standalone Next.js bundle — no source code, no node_modules, no dev deps.
# ────────────────────────────────────────────────────────────────────────────
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Create a non-root user for security best practice
RUN addgroup --system --gid 1001 nodejs \
 && adduser  --system --uid 1001 nextjs

# Copy the standalone server (self-contained, no node_modules needed)
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./

# Copy the static assets (JS/CSS chunks, fonts, etc.)
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# Copy the public folder (favicon, images, etc.)
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# `node server.js` is the entry point for Next.js standalone mode
CMD ["node", "server.js"]
