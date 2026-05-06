FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY shared ./shared
COPY notification-service ./notification-service
WORKDIR /app/notification-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-service ./cmd

FROM alpine:3.20
COPY --from=builder /bin/notification-service /bin/notification-service
ENTRYPOINT ["/bin/notification-service"]
