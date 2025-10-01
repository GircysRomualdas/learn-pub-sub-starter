package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

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

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Failed to welcome client: %v", err)
	}

	queuePauseName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	gameState := gamelogic.NewGameState(username)
	if err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queuePauseName, routing.PauseKey, pubsub.QueueTransient, handlerPause(gameState)); err != nil {
		log.Fatalf("Failed to subscribe to pause queue: %v", err)
	}

	moveKey := fmt.Sprintf("%s.*", routing.ArmyMovesPrefix)
	queueMoveName := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
	if err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, queueMoveName, moveKey, pubsub.QueueTransient, handlerMove(gameState, publishCh)); err != nil {
		log.Fatalf("Failed to subscribe to move queue: %v", err)
	}

	warKey := fmt.Sprintf("%s.*", routing.WarRecognitionsPrefix)
	if err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, warKey, pubsub.QueueDurable, handlerWar(gameState, publishCh)); err != nil {
		log.Fatalf("Failed to subscribe to war queue: %v", err)
	}

	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "spawn":
			err := gameState.CommandSpawn(input)
			if err != nil {
				fmt.Printf("Spawn error: %v\n", err)
			}
		case "move":
			armyMove, err := gameState.CommandMove(input)
			if err != nil {
				fmt.Printf("Move error: %v\n", err)
				continue
			}
			if len(armyMove.Units) == 0 {
				fmt.Printf("No units to move\n")
				continue
			}
			if err := pubsub.PublishJSON(publishCh, routing.ExchangePerilTopic, queueMoveName, armyMove); err != nil {
				fmt.Println("publish error:", err)
				continue
			}
			fmt.Println("Move was published")
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Start spamming")
			if err := spam(publishCh, input, username); err != nil {
				fmt.Printf("Spam error: %v\n", err)
				continue
			}
			fmt.Println("Finished spamming")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Printf("Unknown command: %s\n", input[0])
		}
	}

}

func publishGameLog(publishCh *amqp.Channel, username, msg string) error {
	return pubsub.PublishGob(
		publishCh,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.GameLogSlug, username),
		routing.GameLog{
			Username:    username,
			CurrentTime: time.Now(),
			Message:     msg,
		},
	)
}

func spam(publishCh *amqp.Channel, words []string, username string) error {
	if len(words) != 2 {
		return errors.New("Usage: spam <number>")
	}
	num, err := strconv.Atoi(words[1])
	if err != nil {
		return fmt.Errorf("Error: %s is not a valid number", words[1])
	}

	for i := 0; i < num; i++ {
		msg := gamelogic.GetMaliciousLog()
		gamelog := routing.GameLog{
			Username:    username,
			CurrentTime: time.Now(),
			Message:     msg,
		}
		key := fmt.Sprintf("%s.%s", routing.GameLogSlug, username)
		if err := pubsub.PublishGob(publishCh, routing.ExchangePerilTopic, key, gamelog); err != nil {
			fmt.Println("publish error:", err)
			return fmt.Errorf("Publish error: %v", err)
		}
	}
	return nil
}
