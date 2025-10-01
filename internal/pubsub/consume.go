package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) Acktype) error {
	unmarshaller := func(body []byte) (T, error) {
		var val T
		err := json.Unmarshal(body, &val)
		return val, err
	}
	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
}

func SubscribeGob[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) Acktype) error {
	unmarshaller := func(body []byte) (T, error) {
		var val T
		buffer := bytes.NewBuffer(body)
		decoder := gob.NewDecoder(buffer)
		err := decoder.Decode(&val)
		return val, err
	}
	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
}

func subscribe[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) Acktype, unmarshaller func([]byte) (T, error)) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("Failed to declare and bind queue: %v", err)
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return fmt.Errorf("Failed to set QoS: %v", err)
	}
	deliveries, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("Failed to consume queue: %v", err)
	}

	go func() {
		defer ch.Close()
		for delivery := range deliveries {
			val, err := unmarshaller(delivery.Body)
			if err != nil {
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				continue
			}

			acktype := handler(val)
			switch acktype {
			case Ack:
				delivery.Ack(false)
			case NackRequeue:
				delivery.Nack(false, true)
			case NackDiscard:
				delivery.Nack(false, false)
			default:
				delivery.Nack(false, false)
			}
		}
	}()

	return nil
}
