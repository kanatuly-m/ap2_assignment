FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY shared ./shared
COPY order-service ./order-service
WORKDIR /app/order-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/order-service ./cmd

FROM alpine:3.20
COPY --from=builder /bin/order-service /bin/order-service
EXPOSE 8080
ENTRYPOINT ["/bin/order-service"]
