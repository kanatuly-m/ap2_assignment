package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ap2_assignment/shared/events"
	"notification-service/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	confirms   <-chan amqp.Confirmation
	exchange   string
	routingKey string
	queueName  string
	maxRetries int
	useCase    domain.NotificationUseCase
}

func NewRabbitMQConsumer(url, exchange, routingKey, queueName, dlxName, dlqName string, maxRetries int, uc domain.NotificationUseCase) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := declareTopology(channel, exchange, routingKey, queueName, dlxName, dlqName); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := channel.Qos(1, 0, false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	return &RabbitMQConsumer{
		conn:       conn,
		channel:    channel,
		confirms:   channel.NotifyPublish(make(chan amqp.Confirmation, 1)),
		exchange:   exchange,
		routingKey: routingKey,
		queueName:  queueName,
		maxRetries: maxRetries,
		useCase:    uc,
	}, nil
}

func declareTopology(channel *amqp.Channel, exchange, routingKey, queueName, dlxName, dlqName string) error {
	if err := channel.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.ExchangeDeclare(dlxName, "direct", true, false, false, false, nil); err != nil {
		return err
	}

	_, err := channel.QueueDeclare(queueName, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    dlxName,
		"x-dead-letter-routing-key": routingKey + ".dead",
	})
	if err != nil {
		return err
	}
	if err := channel.QueueBind(queueName, routingKey, exchange, false, nil); err != nil {
		return err
	}

	_, err = channel.QueueDeclare(dlqName, true, false, false, false, nil)
	if err != nil {
		return err
	}
	return channel.QueueBind(dlqName, routingKey+".dead", dlxName, false, nil)
}

func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	deliveries, err := c.channel.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Printf("Notification Service is listening to queue %s", c.queueName)
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}
			c.handleDelivery(ctx, delivery)
		}
	}
}

func (c *RabbitMQConsumer) handleDelivery(ctx context.Context, delivery amqp.Delivery) {
	var event events.PaymentCompletedEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		log.Printf("[Notification] Invalid message moved to DLQ: %v", err)
		_ = delivery.Nack(false, false)
		return
	}

	_, err := c.useCase.HandlePaymentCompleted(ctx, event)
	if err != nil {
		c.handleFailure(ctx, delivery, err)
		return
	}

	if err := delivery.Ack(false); err != nil {
		log.Printf("[Notification] ACK failed: %v", err)
	}
}

func (c *RabbitMQConsumer) handleFailure(ctx context.Context, delivery amqp.Delivery, processingErr error) {
	retryCount := getRetryCount(delivery.Headers)
	if retryCount >= c.maxRetries-1 {
		log.Printf("[Notification] Max attempts reached. Moving message to DLQ. Error: %v", processingErr)
		_ = delivery.Nack(false, false)
		return
	}

	nextRetry := retryCount + 1
	headers := copyHeaders(delivery.Headers)
	headers["x-retry-count"] = int32(nextRetry)

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := c.channel.PublishWithContext(
		publishCtx,
		c.exchange,
		c.routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  delivery.ContentType,
			DeliveryMode: amqp.Persistent,
			MessageId:    delivery.MessageId,
			Timestamp:    time.Now().UTC(),
			Headers:      headers,
			Body:         delivery.Body,
		},
	)
	if err != nil {
		log.Printf("[Notification] Retry publish failed, keeping original message unacked: %v", err)
		_ = delivery.Nack(false, true)
		return
	}

	select {
	case confirmation := <-c.confirms:
		if !confirmation.Ack {
			log.Println("[Notification] Retry publish was not confirmed, requeueing original message")
			_ = delivery.Nack(false, true)
			return
		}
	case <-publishCtx.Done():
		log.Printf("[Notification] Retry publish confirmation timeout: %v", publishCtx.Err())
		_ = delivery.Nack(false, true)
		return
	}

	log.Printf("[Notification] Processing failed. Retry %d/%d scheduled. Error: %v", nextRetry, c.maxRetries, processingErr)
	_ = delivery.Ack(false)
}

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	value, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func copyHeaders(headers amqp.Table) amqp.Table {
	copied := amqp.Table{}
	for key, value := range headers {
		copied[key] = value
	}
	return copied
}

func (c *RabbitMQConsumer) Close() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *RabbitMQConsumer) String() string {
	return fmt.Sprintf("RabbitMQConsumer(queue=%s)", c.queueName)
}
