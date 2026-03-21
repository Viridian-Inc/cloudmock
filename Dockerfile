# Stage 1: Build dashboard
FROM node:20-alpine AS dashboard
WORKDIR /dashboard
COPY dashboard/package*.json ./
RUN npm ci
COPY dashboard/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=dashboard /dashboard/dist/ ./pkg/dashboard/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /cloudmock ./cmd/gateway

# Stage 3: Final image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /cloudmock /usr/local/bin/cloudmock
COPY cloudmock.yml /etc/cloudmock/cloudmock.yml
EXPOSE 4566 4500 4599
ENTRYPOINT ["cloudmock"]
CMD ["--config", "/etc/cloudmock/cloudmock.yml"]
