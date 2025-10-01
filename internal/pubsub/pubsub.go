package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
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

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) Acktype,
) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("Failed to declare and bind queue: %v", err)
	}

	deliveries, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("Failed to consume queue: %v", err)
	}

	go func() {
		defer ch.Close()
		for delivery := range deliveries {
			var val T
			if err := json.Unmarshal(delivery.Body, &val); err != nil {
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				continue
			}
			acktype := handler(val)
			switch acktype {
			case Ack:
				delivery.Ack(false)
				fmt.Println("Message acknowledged")
			case NackRequeue:
				delivery.Nack(false, true)
				fmt.Println("Message requeued")
			case NackDiscard:
				delivery.Nack(false, false)
				fmt.Println("Message discarded")
			default:
				delivery.Nack(false, false)
				fmt.Println("Unknown acktype")
			}
		}
	}()

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(val); err != nil {
		return err
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body:        buffer.Bytes(),
	})
}
