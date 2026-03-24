# Stage 1: Build dashboard
FROM node:20-alpine AS dashboard
WORKDIR /app/dashboard
COPY dashboard/package.json dashboard/package-lock.json ./
RUN npm ci
COPY dashboard/ .
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=dashboard /app/dashboard/dist ./pkg/dashboard/dist/
RUN CGO_ENABLED=1 go build -ldflags="-s -w -X github.com/neureaux/cloudmock/pkg/admin.Version=1.0.0" -o /app/bin/gateway ./cmd/gateway

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /app/bin/gateway /app/gateway
COPY cloudmock.yml /app/cloudmock.yml
WORKDIR /app
EXPOSE 4566 4500 4599
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q --spider http://localhost:4599/api/health || exit 1
ENTRYPOINT ["/app/gateway"]
