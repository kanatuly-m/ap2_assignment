CREATE DATABASE order_db;
CREATE DATABASE payment_db;

\c order_db;
CREATE TABLE orders (
                        id TEXT PRIMARY KEY,
                        customer_id TEXT NOT NULL,
                        item_name TEXT NOT NULL,
                        amount BIGINT NOT NULL,
                        status TEXT NOT NULL,
                        created_at TIMESTAMP NOT NULL,
                        idempotency_key TEXT UNIQUE
);

\c payment_db;
CREATE TABLE payments (
                          id TEXT PRIMARY KEY,
                          order_id TEXT NOT NULL UNIQUE,
                          transaction_id TEXT NOT NULL,
                          amount BIGINT NOT NULL,
                          status TEXT NOT NULL
);