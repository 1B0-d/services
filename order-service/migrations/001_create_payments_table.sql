CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL,
    customer_email VARCHAR(255) NOT NULL DEFAULT '',
    item_name VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL
);

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS customer_email VARCHAR(255) NOT NULL DEFAULT '';
