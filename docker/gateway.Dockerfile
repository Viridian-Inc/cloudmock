FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gateway ./cmd/gateway

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /gateway /app/gateway
COPY cloudmock.yml /app/cloudmock.yml
EXPOSE 4566 4500 4599
ENTRYPOINT ["/app/gateway"]
CMD ["--config", "/app/cloudmock.yml"]
