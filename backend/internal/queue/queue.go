// Package queue is a thin wrapper around a single RabbitMQ topic exchange
// ("trip.events"). It's intentionally minimal for the MVP: one exchange, a
// couple of routing keys (trip.started, trip.eta_updated), no retry/DLQ
// topology - see SPECIFIKACIJA.md section 3.6 for the scope reasoning.
package queue

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "trip.events"

type Client struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func Connect(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	return &Client{conn: conn, ch: ch}, nil
}

func (c *Client) Close() {
	c.ch.Close()
	c.conn.Close()
}

func (c *Client) Publish(ctx context.Context, routingKey string, body []byte) error {
	return c.ch.PublishWithContext(ctx, Exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// Consume declares a durable queue bound to routingKey on the shared exchange and
// returns the delivery channel. queueName should be unique per consumer role.
func (c *Client) Consume(queueName, routingKey string) (<-chan amqp.Delivery, error) {
	q, err := c.ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	if err := c.ch.QueueBind(q.Name, routingKey, Exchange, false, nil); err != nil {
		return nil, fmt.Errorf("bind queue: %w", err)
	}

	deliveries, err := c.ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	return deliveries, nil
}
