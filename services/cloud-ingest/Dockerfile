FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /ingest ./cmd/ingest

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /ingest /ingest
COPY --from=builder /app/migrations /migrations
EXPOSE 8080
CMD ["/ingest"]
