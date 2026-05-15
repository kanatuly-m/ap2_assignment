FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY shared ./shared
COPY payment-service ./payment-service
WORKDIR /app/payment-service
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/payment-service ./cmd

FROM alpine:3.20
COPY --from=builder /bin/payment-service /bin/payment-service
EXPOSE 8081
ENTRYPOINT ["/bin/payment-service"]
