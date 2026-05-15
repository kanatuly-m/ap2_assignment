module notification-service

go 1.25

require (
	ap2_assignment/shared v0.0.0
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/redis/go-redis/v9 v9.18.0
)

replace ap2_assignment/shared => ../shared
