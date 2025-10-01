package pubsub

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string

const (
	QueueDurable   SimpleQueueType = "durable"
	QueueTransient SimpleQueueType = "transient"
)

type Acktype int

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
)

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

	args := amqp.Table{
		"x-dead-letter-exchange": routing.ExchangePerilDLX,
	}

	queue, err := ch.QueueDeclare(queueName, isDurable, isTransient, isTransient, false, args)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to declare queue: %v", err)
	}

	if err := ch.QueueBind(queue.Name, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to bind queue: %v", err)
	}

	return ch, queue, nil
}
