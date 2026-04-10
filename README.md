# AP2 Assignment 1 — Clean Architecture based Microservices (Order & Payment)

## Overview

This project implements a two-service platform in Go:

- **Order Service**
- **Payment Service**

The system follows **Clean Architecture** and **microservice decomposition**.
Each service has its own responsibility, its own data, and its own database.
The communication between services is implemented using **REST** with **Gin** and a custom HTTP client timeout.

---

## Services

### 1. Order Service

Responsible for:

- creating orders
- storing orders in its own database
- calling Payment Service through HTTP
- updating order status based on payment result
- returning order details
- cancelling only pending orders

Supported endpoints:

- `POST /orders`
- `GET /orders/{id}`
- `PATCH /orders/{id}/cancel`

### 2. Payment Service

Responsible for:

- processing payment requests
- validating payment limits
- storing payment information in its own database
- returning payment status for an order

Supported endpoints:

- `POST /payments`
- `GET /payments/{order_id}`

---

## Clean Architecture

Each service is organized using the following layers:

- **domain** — entities, business interfaces, status constants
- **usecase** — business logic and state transitions
- **repository** — persistence logic and outbound HTTP client implementation
- **transport/http** — handlers, request parsing, response formatting, routes
- **cmd/.../main.go** — composition root and manual dependency injection

This means:

- handlers stay thin
- business logic lives in use cases
- persistence lives in repository layer
- domain models do not depend on HTTP or framework code
- use cases depend on interfaces (ports)

---

## Bounded Contexts and Data Ownership

This system is decomposed into **two bounded contexts**:

### Order Context
Owns:
- order creation
- order lifecycle
- order statuses (`Pending`, `Paid`, `Failed`, `Cancelled`)

### Payment Context
Owns:
- payment authorization
- transaction limits
- payment records
- payment statuses (`Authorized`, `Declined`)

### Data ownership
Each service has its **own database** and its **own internal models**.

- `order-service` uses the `orders` table
- `payment-service` uses the `payments` table

There is **no shared database** and **no shared common entity package**.

---

## Architecture Diagram

Client
  |
  v
Order Service HTTP Handlers
  |
  v
Order Use Case
  |
  +----> Order Repository ----> Order DB
  |
  +----> Payment HTTP Client ----> Payment Service HTTP Handlers
                                      |
                                      v
                                 Payment Use Case
                                      |
                                      v
                               Payment Repository ----> Payment DB

### Dependency flow inside each service

main.go / Composition Root
  -> transport/http
  -> usecase
  -> interfaces / ports
  -> repository
  -> database

---

## Domain Models

### Order

- `ID`
- `CustomerID`
- `ItemName`
- `Amount int64`
- `Status`
- `CreatedAt`

Order statuses:
- `Pending`
- `Paid`
- `Failed`
- `Cancelled`

### Payment

- `ID`
- `OrderID`
- `TransactionID`
- `Amount int64`
- `Status`

Payment statuses:
- `Authorized`
- `Declined`

Money is stored as `int64`, not `float64`.

---

## Business Rules

### Order rules
- amount must be greater than `0`
- `Paid` orders cannot be cancelled
- only `Pending` orders can be cancelled

### Payment rules
- if `amount > 100000`, Payment Service returns `Declined`
- otherwise Payment Service returns `Authorized`

### Service interaction rule
- Order Service uses a custom `http.Client` with timeout of **2 seconds**

---

## Order Flow

### Successful payment
1. Client sends `POST /orders`
2. Order Service creates a new order with status `Pending`
3. Order Service stores the order in Order DB
4. Order Service calls `POST /payments`
5. Payment Service authorizes the payment
6. Order Service updates the order status to `Paid`
7. Client receives order response with status `Paid`

### Declined payment
1. Order is created as `Pending`
2. Payment Service returns `Declined`
3. Order Service updates order status to `Failed`

### Payment service unavailable
1. Order is created as `Pending`
2. Payment Service call fails or times out
3. Order Service returns **503 Service Unavailable**
4. The order remains **Pending**

I chose **Pending** for the payment-unavailable scenario because the order was already created, but payment confirmation was not received due to an external service failure.

---

## Why this design was chosen

### Separate databases
This design avoids tight coupling and follows the **database-per-service** rule.
Each service is responsible only for its own data.

### No shared code
Each service has its own models and internal packages, which helps preserve service boundaries.

### Thin handlers
Handlers only:
- parse input
- call use case
- return HTTP response

Business rules are not placed inside handlers.

### Manual dependency injection
All dependencies are wired in `main.go`, which acts as the composition root.

### Outbound payment client as a port
Order Use Case depends on an abstraction for payment calls rather than HTTP details directly.
This keeps the business logic cleaner and easier to extend or test.

---

## Project Structure

order-service/
├── cmd/order-service/main.go
├── internal/
│   ├── domain/
│   ├── usecase/
│   ├── repository/
│   └── transport/http/
├── migrations/
│   └── 001_create_orders_table.sql
└── go.mod

payment-service/
├── cmd/payment-service/main.go
├── internal/
│   ├── domain/
│   ├── usecase/
│   ├── repository/
│   └── transport/http/
├── migrations/
│   └── 001_create_payments_table.sql
└── go.mod

---

## Database Schema / Migrations

### Order Service migration
File: `order-service/migrations/001_create_orders_table.sql`

    CREATE TABLE IF NOT EXISTS orders (
        id UUID PRIMARY KEY,
        customer_id VARCHAR(255) NOT NULL,
        item_name VARCHAR(255) NOT NULL,
        amount BIGINT NOT NULL,
        status VARCHAR(50) NOT NULL,
        created_at TIMESTAMP NOT NULL
    );

### Payment Service migration
File: `payment-service/migrations/001_create_payments_table.sql`

    CREATE TABLE IF NOT EXISTS payments (
        id UUID PRIMARY KEY,
        order_id VARCHAR(255) NOT NULL UNIQUE,
        transaction_id VARCHAR(255),
        amount BIGINT NOT NULL,
        status VARCHAR(50) NOT NULL
    );

---

## How to Run

### 1. Start databases

Use Docker Compose from the project root:

    docker compose up -d

### 2. Create database tables

Run the SQL files inside each service database.

### 3. Start Payment Service

    cd payment-service
    go run ./cmd/payment-service

### 4. Start Order Service

    cd order-service
    go run ./cmd/order-service

---

## API Examples

### Create payment

    curl -X POST http://localhost:8081/payments \
      -H "Content-Type: application/json" \
      -d "{\"order_id\":\"order-1\",\"amount\":15000}"

### Get payment by order id

    curl http://localhost:8081/payments/order-1

### Create order

    curl -X POST http://localhost:8080/orders \
      -H "Content-Type: application/json" \
      -d "{\"customer_id\":\"cust-1\",\"item_name\":\"Laptop\",\"amount\":15000}"

### Get order by id

    curl http://localhost:8080/orders/{id}

### Cancel pending order

    curl -X PATCH http://localhost:8080/orders/{id}/cancel

---

## Tested Scenarios

### Payment success
- input amount within limit
- payment status becomes `Authorized`
- order status becomes `Paid`

### Payment declined
- input amount greater than `100000`
- payment status becomes `Declined`
- order status becomes `Failed`

### Payment service unavailable
- order service does not hang
- timeout trips
- order service returns `503 Service Unavailable`
- order remains `Pending`

### Cancel order
- `Pending` order can be cancelled
- `Paid` order cannot be cancelled

---

## Trade-offs

- I used synchronous REST communication because it is explicitly required by the assignment.
- I kept the services small and focused on one responsibility each.
- I chose **Pending** for the payment-unavailable scenario because the order was already created, but payment confirmation was not received.
- A possible future improvement would be adding idempotency using an `Idempotency-Key`.

---

## Submission Contents

This submission includes:

- source code for both services
- SQL migration/schema files
- this README
- architecture diagram embedded in README
- curl API examples
"# services" 
