# Stage 1: Build devtools UI
FROM node:22-alpine AS dashboard
# Pin pnpm to v9 to match the release pipeline's build-devtools job
# (pnpm/action-setup version: 9) and the lockfileVersion 9.0 lockfile.
# Using pnpm@latest here let a newer major (pnpm 10+, with stricter
# build-script/supply-chain gating) break `pnpm install --frozen-lockfile`.
RUN corepack enable && corepack prepare pnpm@9 --activate
WORKDIR /devtools
COPY devtools/package.json devtools/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY devtools/ ./
RUN pnpm build

# Stage 2: Build Go binary
FROM golang:1.26-bookworm AS builder
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=dashboard /devtools/dist/ ./pkg/dashboard/dist/
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /cloudmock ./cmd/gateway

# Stage 3: Final image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /cloudmock /usr/local/bin/cloudmock
COPY cloudmock.yml /etc/cloudmock/cloudmock.yml
EXPOSE 4566 4500 4599
ENTRYPOINT ["cloudmock"]
CMD ["--config", "/etc/cloudmock/cloudmock.yml"]
