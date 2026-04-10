CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL UNIQUE,
    transaction_id VARCHAR(255),
    amount BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL
);