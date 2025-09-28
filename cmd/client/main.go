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

	queuePauseName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	gameState := gamelogic.NewGameState(username)
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queuePauseName, routing.PauseKey, pubsub.QueueTransient, handlerPause(gameState))
	if err != nil {
		fmt.Printf("Failed to subscribe to pause queue: %v\n", err)
		return
	}

	moveKey := fmt.Sprintf("%s.*", routing.ArmyMovesPrefix)
	queueMoveName := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, queueMoveName, moveKey, pubsub.QueueTransient, handlerMove(gameState))
	if err != nil {
		fmt.Printf("Failed to subscribe to move queue: %v\n", err)
		return
	}

	key := fmt.Sprintf("%s.*", routing.GameLogSlug)
	ch, _, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, key, pubsub.QueueDurable)
	if err != nil {
		fmt.Printf("Failed to declare and bind queue: %v\n", err)
		return
	}
	defer ch.Close()

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
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, queueMoveName, armyMove); err != nil {
				fmt.Println("publish error:", err)
				continue
			}
			fmt.Println("Move was published")
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(move gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)
		fmt.Println("Detected move:", outcome)
	}
}
