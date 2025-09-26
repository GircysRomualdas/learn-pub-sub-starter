package main

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {
	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		fmt.Printf("Failed to connect to RabbitMQ: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to RabbitMQ")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Printf("Failed to welcome client: %v\n", err)
		return
	}

	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	ch, _, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.QueueTransient)
	if err != nil {
		fmt.Printf("Failed to declare and bind queue: %v\n", err)
		return
	}
	defer ch.Close()

	gameState := gamelogic.NewGameState(username)
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
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Printf("Unknown command: %s\n", input[0])
		}
	}

}
