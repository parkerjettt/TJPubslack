package main

import (
	"context"
	"flag"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
)

var (
	topic = flag.String("topic", "", "Topic name for publishing")
)

func main() {
	flag.Parse()
	projectId := "alphaus-live"
	ctx := context.Background()

	if *topic == "" {
		log.Println("topic cannot be empty")
		return
	}

	client, err := pubsub.NewClient(ctx, projectId)
	if err != nil {
		log.Println("NewClient failed:", err)
		return
	}

	defer client.Close()
	t := client.Topic(*topic)

	// Define your messages
	messages := []string{
		`{
			"date":"2024-01-01",
			"service":"AmazonEC2",
			"description":"This is a sample description.",
			"cost":1234.56
		}`,
		`{
			"date":"2024-01-02",
			"service":"Google Cloud Storage",
			"description":"Another sample description.",
			"cost":789.10
		}`,
		// Add more messages as needed
	}

	// Publish each message
	for _, msgData := range messages {
		result := t.Publish(ctx, &pubsub.Message{
			Data: []byte(msgData),
		})

		// Block until the result is returned and a server-generated
		// ID is returned for the published message.
		id, err := result.Get(ctx)
		if err != nil {
			log.Println("Get failed:", err)
			continue
		}

		log.Printf("Published message with ID: %v\n", id)

		// Sleep for a short duration to avoid flooding Pub/Sub
		time.Sleep(500 * time.Millisecond)
	}
}
