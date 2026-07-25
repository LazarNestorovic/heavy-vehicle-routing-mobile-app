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

// ChatExchange carries chat messages, routed by conversation (see ChatRoutingKey)
// rather than by event type - each chat WS connection binds its own ephemeral
// queue to the routing key for its conversation instead of sharing a durable
// named queue like the trip.events consumers do (see ConsumeChatEphemeral).
const ChatExchange = "chat.events"

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

	if err := ch.ExchangeDeclare(ChatExchange, "topic", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare chat exchange: %w", err)
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

func (c *Client) PublishChat(ctx context.Context, routingKey string, body []byte) error {
	return c.ch.PublishWithContext(ctx, ChatExchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// ConsumeChatEphemeral binds a throwaway, exclusive, auto-delete queue to routingKey
// on ChatExchange - one per live chat WS connection. Unlike Consume's durable named
// queues (meant for a persistent worker subscription), a chat WS client that isn't
// connected has nothing to catch up on via RabbitMQ - REST (ListThread) is the
// durable channel, this is purely for live delivery while connected. It opens its
// own amqp.Channel (a Channel isn't safe for concurrent use across goroutines, and
// this runs independently of the shared publisher channel). The returned close
// func must be called exactly once to release the channel.
func (c *Client) ConsumeChatEphemeral(routingKey string) (<-chan amqp.Delivery, func(), error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("open channel: %w", err)
	}

	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		ch.Close()
		return nil, nil, fmt.Errorf("declare queue: %w", err)
	}

	if err := ch.QueueBind(q.Name, routingKey, ChatExchange, false, nil); err != nil {
		ch.Close()
		return nil, nil, fmt.Errorf("bind queue: %w", err)
	}

	deliveries, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		ch.Close()
		return nil, nil, fmt.Errorf("consume: %w", err)
	}
	return deliveries, func() { ch.Close() }, nil
}
