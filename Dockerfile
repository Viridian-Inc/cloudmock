# Stage 1: Build devtools UI
FROM node:22-alpine AS dashboard
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /devtools
COPY devtools/package.json devtools/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY devtools/ ./
RUN pnpm build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=dashboard /devtools/dist/ ./pkg/dashboard/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /cloudmock ./cmd/gateway

# Stage 3: Final image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /cloudmock /usr/local/bin/cloudmock
COPY cloudmock.yml /etc/cloudmock/cloudmock.yml
EXPOSE 4566 4500 4599
ENTRYPOINT ["cloudmock"]
CMD ["--config", "/etc/cloudmock/cloudmock.yml"]
