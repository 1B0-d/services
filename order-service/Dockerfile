FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY ap-pb ./ap-pb
COPY AP_assik1/order-service/go.mod AP_assik1/order-service/go.sum ./AP_assik1/order-service/

WORKDIR /src/AP_assik1/order-service
RUN go mod download

COPY AP_assik1/order-service ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/order-service ./cmd/order-service

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /out/order-service ./order-service
COPY AP_assik1/order-service/migrations ./migrations

EXPOSE 8080 50052
ENTRYPOINT ["/app/order-service"]
