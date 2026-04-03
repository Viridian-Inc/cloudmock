FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY main.go .
RUN go build -o notification-service .

FROM alpine:3.19
COPY --from=build /app/notification-service /notification-service
CMD ["/notification-service"]
