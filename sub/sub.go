package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"cloud.google.com/go/pubsub"
)

var (
	subId           = flag.String("subscription", "", "Subscription name")
	slackWebhookURL = "https://hooks.slack.com/services/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"

	// Variables to store the state
	totalCost         float64
	totalCostPerDate  map[string]float64
	totalMessages     int
	totalCostToDate   float64
	averageCostToDate float64
	mutex             sync.Mutex
)

type SlackMessage struct {
	Text string `json:"text"`
}

type Message struct {
	Date        string  `json:"date"`
	Service     string  `json:"service"`
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
}

func main() {
	flag.Parse()
	projectId := "alphaus-live"
	ctx := context.Background()

	if *subId == "" {
		log.Println("subscription cannot be empty")
		return
	}

	// Initialize state variables
	totalCostPerDate = make(map[string]float64)

	client, err := pubsub.NewClient(ctx, projectId)
	if err != nil {
		log.Println("NewClient failed:", err)
		return
	}

	defer client.Close()
	sub := client.Subscription(*subId)
	sub.ReceiveSettings.Synchronous = true

	// Receive blocks until the context is cancelled or an error occurs.
	err = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		log.Printf("Received: %q", msg)
		processMessage(msg.Data)
		msg.Ack()
	})

	if err != nil {
		log.Println("Receive failed:", err)
		return
	}
}

func processMessage(data []byte) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		log.Println("Error unmarshalling message:", err)
		return
	}

	// Update the state variables in a thread-safe manner
	mutex.Lock()
	defer mutex.Unlock()

	totalMessages++
	totalCost += msg.Cost
	totalCostToDate += msg.Cost
	totalCostPerDate[msg.Date] += msg.Cost
	averageCostToDate = totalCostToDate / float64(totalMessages)

	// Prepare the Slack message
	slackMsg := prepareSlackMessage(msg)
	sendToSlack(slackMsg)
}

func prepareSlackMessage(msg Message) string {
	slackMsg := SlackMessage{
		Text: "New message received:\n" +
			"Date: " + msg.Date + "\n" +
			"Service: " + msg.Service + "\n" +
			"Description: " + msg.Description + "\n" +
			"Cost: " + formatCost(msg.Cost) + "\n" +
			"Total Cost: " + formatCost(totalCost) + "\n" +
			"Total Cost Per Date: " + formatCostPerDate() + "\n" +
			"Total Cost To Date: " + formatCost(totalCostToDate) + "\n" +
			"Average Cost To Date: " + formatCost(averageCostToDate) + "\n" +
			"Number of Messages Processed So Far: " + formatNumber(totalMessages),
	}

	slackPayload, err := json.Marshal(slackMsg)
	if err != nil {
		log.Println("Error marshalling Slack message:", err)
		return ""
	}

	return string(slackPayload)
}

func formatCost(cost float64) string {
	return "$" + strconv.FormatFloat(cost, 'f', 2, 64)
}

func formatCostPerDate() string {
	var builder strings.Builder
	builder.WriteString("[")
	for date, cost := range totalCostPerDate {
		builder.WriteString(date)
		builder.WriteString(": ")
		builder.WriteString(formatCost(cost))
		builder.WriteString(", ")
	}
	builder.WriteString("]")
	return builder.String()
}

func formatNumber(num int) string {
	return strconv.Itoa(num)
}

func sendToSlack(message string) {
	resp, err := http.Post(slackWebhookURL, "application/json", strings.NewReader(message))
	if err != nil {
		log.Println("Error sending message to Slack:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Non-OK status code received from Slack:", resp.StatusCode)
	}
}
