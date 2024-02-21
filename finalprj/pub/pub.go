package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
)

var (
	topicID = flag.String("topic", "trial-L", "Topic name for publishing")
)

type Message struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Service     string  `json:"service"`
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
}

func main() {
	flag.Parse()
	projectId := "alphaus-live"
	ctx := context.Background()

	if *topicID == "" {
		log.Println("topic cannot be empty")
		return
	}

	// Create a Google Cloud Pub/Sub client
	client, err := pubsub.NewClient(ctx, projectId)
	if err != nil {
		log.Println("NewClient failed:", err)
		return
	}
	defer client.Close()

	// Create a Google Cloud Pub/Sub topic
	topic := client.Topic(*topicID)

	// Listen for signals to gracefully stop message publishing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start publishing messages in a loop
	for {
		select {
		case <-stop:
			log.Println("Received interrupt signal. Stopping message publishing.")
			return
		default:
			msg := generateMessage()
			data, err := json.Marshal(msg)
			if err != nil {
				log.Println("Error marshalling message:", err)
				continue
			}

			result := topic.Publish(ctx, &pubsub.Message{
				Data: data,
			})

			id, err := result.Get(ctx)
			if err != nil {
				log.Println("Get failed:", err)
				continue
			}

			log.Printf("Published message with ID: %v\n", id)

		}
	}
}

func generateMessage() Message {
	return Message{
		ID:          generateUniqueID(),
		Date:        randomDate(),
		Service:     randomService(),
		Description: randomDescription(),
		Cost:        randomCost(),
	}
}

// randomDate generates a random date string in the format "2006-01-02".
func randomDate() string {
	// Define a range of dates suitable for your application.
	// Here, I'm assuming a range of 30 days.
	min := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2023, 1, 30, 0, 0, 0, 0, time.UTC).Unix()

	delta := max - min
	sec := rand.Int63n(delta) + min
	randomTime := time.Unix(sec, 0)
	return randomTime.Format("2006-01-02")
}

// randomService generates a random service name.
func randomService() string {
	services := []string{"Service A", "Service B", "Service C"}
	return services[rand.Intn(len(services))]
}

// randomDescription generates a random description.
func randomDescription() string {
	descriptions := []string{"Description A", "Description B", "Description C"}
	return descriptions[rand.Intn(len(descriptions))]
}

// randomCost generates a random cost.
func randomCost() float64 {
	// Generate a random cost between 0 and 100.
	return rand.Float64() * 100
}

func generateUniqueID() string {
	// Implement your logic to generate a unique ID here
	return time.Now().Format("20060102150405") // Example: timestamp as ID
}
