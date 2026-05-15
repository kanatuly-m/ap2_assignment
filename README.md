# AP2 Assignment 4 — Performance Optimization & External Integrations

This project builds on the previous microservices system:

- **Order Service** — REST API for orders.
- **Payment Service** — processes payments and publishes RabbitMQ events.
- **Notification Service** — background worker consuming RabbitMQ events.
- **PostgreSQL** — stores orders and payments.
- **RabbitMQ** — event broker.
- **Redis** — cache, background-job idempotency, and API rate limiting.

## What was added in Assignment 4

### 1. Redis cache-aside in Order Service

`GET /orders/:id` now uses the cache-aside pattern:

1. Check Redis key `order:{id}`.
2. On cache HIT, return cached JSON.
3. On cache MISS, read PostgreSQL, cache the result, and return it.
4. TTL is configured through `.env` using `CACHE_TTL_SECONDS`.

Default TTL:

```env
CACHE_TTL_SECONDS=300
```

### 2. Cache invalidation

When an order status changes, the matching Redis key is removed immediately:

- after payment updates the order to `Paid` or `Failed`;
- after a cancellation update.

This prevents stale order statuses from being returned.

Example logs:

```text
[Cache] MISS order:...
[Cache] SET order:... ttl=5m0s
[Cache] HIT order:...
[Cache] INVALIDATED order:... reason=order status changed after payment
```

### 3. Provider Adapter in Notification Service

The worker depends on an `EmailSender` interface, not on a concrete provider.

Supported modes:

- `PROVIDER_MODE=SIMULATED` — default demo adapter. It adds latency and random provider failures.
- `PROVIDER_MODE=REAL` — SMTP adapter using environment variables.

The simulated provider is configured in `.env`:

```env
PROVIDER_MODE=SIMULATED
SIMULATED_LATENCY_MS=500
SIMULATED_FAILURE_RATE=0.20
SIMULATED_ALWAYS_FAIL_EMAIL=fail@example.com
```

`fail@example.com` is intentionally used to demonstrate retries and DLQ movement.

### 4. Reliable background jobs

Notification Service remains fully asynchronous:

```text
Payment Service -> RabbitMQ -> Notification Worker -> Provider Adapter
```

The HTTP response does **not** wait for a slow external notification provider.

### 5. Redis idempotency record for notification jobs

Before sending, the worker checks Redis key:

```text
notification:payment:{payment_id}
```

Statuses are stored in Redis:

- `processing`
- `failed`
- `sent`

If the same `payment_id` is published again and status is already `sent`, the worker skips duplicate sending.

### 6. Exponential backoff

When the provider fails, the RabbitMQ worker retries with increasing delays:

```env
MAX_RETRIES=3
RETRY_BASE_DELAY_SECONDS=2
```

Default timing:

- Retry 1 after 2 seconds
- Retry 2 after 4 seconds
- Retry 3 after 8 seconds
- Next failure moves the message to DLQ

### 7. DLQ from Assignment 3 is preserved

Failed messages move to:

```text
payment.completed.dlq
```

### 8. Bonus: Redis API rate limiter

Order Service includes Gin middleware that stores counters in Redis:

```text
rate:{client_ip}
```

Default limit:

```env
RATE_LIMIT=10
RATE_WINDOW_SECONDS=60
```

When the limit is exceeded, the API returns:

```http
429 Too Many Requests
```

## Clean boundary design

The use cases do not directly initialize Redis, HTTP clients, RabbitMQ, or SMTP.

- Order Use Case depends on interfaces:
  - `OrderRepository`
  - `OrderCache`
  - `PaymentGateway`
- Notification Use Case depends on interfaces:
  - `NotificationJobStore`
  - `EmailSender`

Concrete implementations are placed in infrastructure/adapter packages.

## Run the project

From the project root:

```bash
docker compose down -v --remove-orphans
docker compose up --build
```

Main endpoints:

- Order Service: `http://localhost:8080`
- Payment Service: `http://localhost:8081`
- RabbitMQ UI: `http://localhost:15672`
  - username: `guest`
  - password: `guest`
- Redis: `localhost:6379`

## Demo 1 — Create an order

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: assignment4-order-001" \
  -d '{
    "customer_id": "123",
    "customer_email": "user@example.com",
    "item_name": "Coffee",
    "amount": 9999
  }'
```

Expected result: order status becomes `Paid`, Payment Service publishes an event, Notification Worker processes it asynchronously.

## Demo 2 — Cache MISS and HIT

Copy the returned order ID and run twice:

```bash
curl http://localhost:8080/orders/ORDER_ID
curl http://localhost:8080/orders/ORDER_ID
```

Then view logs:

```bash
docker logs ap2_order_service
```

Expected logs:

```text
[Cache] MISS order:ORDER_ID
[Cache] SET order:ORDER_ID ttl=5m0s
[Cache] HIT order:ORDER_ID
```

## Demo 3 — Duplicate notification prevention via Redis

Call Payment Service again with the same order ID. Payment Service republishes the same payment event, and Notification Worker should skip duplicate sending because Redis already stores status `sent`.

```bash
curl -X POST http://localhost:8081/payments \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "ORDER_ID",
    "customer_email": "user@example.com",
    "amount": 9999
  }'
```

Check:

```bash
docker logs ap2_notification_service
```

Expected duplicate log:

```text
[Notification] Duplicate payment job skipped: payment_id=...
```

## Demo 4 — Retry + exponential backoff + DLQ

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: assignment4-dlq-001" \
  -d '{
    "customer_id": "123",
    "customer_email": "fail@example.com",
    "item_name": "Coffee",
    "amount": 9999
  }'
```

Check logs:

```bash
docker logs -f ap2_notification_service
```

Expected messages:

```text
Retry 1/3 in 2s
Retry 2/3 in 4s
Retry 3/3 in 8s
Max retries exhausted. Moving message to DLQ.
```

Open RabbitMQ UI and check queue:

```text
payment.completed.dlq
```

## Demo 5 — Rate limiter bonus

```bash
for i in $(seq 1 12); do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/orders/unknown-id
done
```

Expected: after the configured threshold, responses become `429`.

## Real email mode

The project includes an SMTP adapter. Keep `PROVIDER_MODE=SIMULATED` for the assignment demo. To use real email sending, set:

```env
PROVIDER_MODE=REAL
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@example.com
SMTP_PASSWORD=your_app_password
SMTP_FROM=your_email@example.com
```

Do not publish real credentials in a public repository.

## Submission checklist

- Source code for Order, Payment, Notification services
- Redis cache-aside + TTL + invalidation
- Background worker + Adapter Pattern
- Redis-based job idempotency
- Exponential backoff retries
- Docker Compose with Redis
- `.env` configuration
- Updated architecture diagram
- README explanation
- Bonus Redis rate limiter
