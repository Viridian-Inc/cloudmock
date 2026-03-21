FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /s3-service ./services/s3/cmd

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /s3-service /app/s3-service
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s \
    CMD wget -qO- http://localhost:8080/_health || exit 1
ENTRYPOINT ["/app/s3-service"]
