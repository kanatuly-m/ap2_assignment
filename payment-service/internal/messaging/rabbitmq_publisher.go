package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ap2_assignment/shared/events"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	confirms   <-chan amqp.Confirmation
	exchange   string
	routingKey string
}

func NewRabbitMQPublisher(url, exchange, routingKey, queueName, dlxName, dlqName string) (*RabbitMQPublisher, error) {
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

	return &RabbitMQPublisher{
		conn:       conn,
		channel:    channel,
		confirms:   channel.NotifyPublish(make(chan amqp.Confirmation, 1)),
		exchange:   exchange,
		routingKey: routingKey,
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

func (p *RabbitMQPublisher) PublishPaymentCompleted(ctx context.Context, event events.PaymentCompletedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		publishCtx,
		p.exchange,
		p.routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.EventID,
			Timestamp:    event.CreatedAt,
			Body:         body,
		},
	)
	if err != nil {
		return err
	}

	select {
	case confirmation := <-p.confirms:
		if !confirmation.Ack {
			return errors.New("rabbitmq did not confirm the published event")
		}
		return nil
	case <-publishCtx.Done():
		return fmt.Errorf("rabbitmq publish confirmation timeout: %w", publishCtx.Err())
	}
}

func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
