module notification-service

go 1.25

require (
	ap2_assignment/shared v0.0.0
	github.com/lib/pq v1.12.3
	github.com/rabbitmq/amqp091-go v1.10.0
)

replace ap2_assignment/shared => ../shared
