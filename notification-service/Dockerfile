FROM golang:1.25-alpine AS builder

WORKDIR /src/AP_assik1/notification-service
COPY AP_assik1/notification-service/go.mod AP_assik1/notification-service/go.sum ./
RUN go mod download

COPY AP_assik1/notification-service ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notification-service ./cmd/notification-service

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /out/notification-service ./notification-service

ENTRYPOINT ["/app/notification-service"]
