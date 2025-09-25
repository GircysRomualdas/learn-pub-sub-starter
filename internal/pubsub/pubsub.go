package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string

const (
	QueueDurable   SimpleQueueType = "durable"
	QueueTransient SimpleQueueType = "transient"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	body, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to open a channel: %v", err)
	}

	isDurable := queueType == QueueDurable
	isTransient := queueType == QueueTransient

	queue, err := ch.QueueDeclare(queueName, isDurable, isTransient, isTransient, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to declare queue: %v", err)
	}

	if err := ch.QueueBind(queue.Name, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to bind queue: %v", err)
	}

	return ch, queue, nil
}
