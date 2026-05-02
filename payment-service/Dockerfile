FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY ap-pb ./ap-pb
COPY AP_assik1/payment-service/go.mod AP_assik1/payment-service/go.sum ./AP_assik1/payment-service/

WORKDIR /src/AP_assik1/payment-service
RUN go mod download

COPY AP_assik1/payment-service ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/payment-service ./cmd/payment-service

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /out/payment-service ./payment-service
COPY AP_assik1/payment-service/migrations ./migrations

EXPOSE 8081 50051
ENTRYPOINT ["/app/payment-service"]
