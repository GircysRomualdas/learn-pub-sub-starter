package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {
	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}
	defer ch.Close()

	logKey := fmt.Sprintf("%s.*", routing.GameLogSlug)
	if err = pubsub.SubscribeGob(conn, routing.ExchangePerilTopic, routing.GameLogSlug, logKey, pubsub.QueueDurable, handlerLog()); err != nil {
		log.Fatalf("Failed to subscribe to log queue: %v", err)
	}

	gamelogic.PrintServerHelp()
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "pause":
			fmt.Println("Sending pause message…")
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true}); err != nil {
				fmt.Println("publish error:", err)
			}
		case "resume":
			fmt.Println("Sending resume message…")
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false}); err != nil {
				fmt.Println("publish error:", err)
			}
		case "quit":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Unknown command:", input[0])
		}
	}
}
