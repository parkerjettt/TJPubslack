package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Message struct {
	ID          int64            `json:"id"`
	Date        spanner.NullDate `json:"date"`
	Service     string           `json:"service"`
	Description string           `json:"description"`
	Cost        float64          `json:"cost"`
}

func main() {
	// Fetch data from Spanner
	projectId := "alphaus-live"
	ctx := context.Background()

	spannerClient, err := spanner.NewClient(ctx, "projects/"+projectId+"/instances/intern2024ft/databases/default", option.WithCredentialsFile(`D:\Alp\internship202401svcacct.json`))
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer spannerClient.Close()

	messages, err := fetchData(ctx, spannerClient)
	if err != nil {
		log.Fatalf("Error fetching data: %v", err)
	}

	// Calculate analytics
	runningTotalCostToDate := calculateRunningTotalCost(messages)
	runningTotalCostPerDate := calculateRunningTotalCostPerDate(messages)
	runningAverageCostToDate := calculateRunningAverageCost(messages)
	messagesProcessedSoFar := len(messages)

	// Construct the message
	output := strings.Builder{}
	output.WriteString("```")
	output.WriteString("Running Total Cost to Date: ")
	output.WriteString(fmt.Sprintf("%.2f\n", runningTotalCostToDate))
	output.WriteString("Running Total Cost Per Date:\n")
	for dateStr, cost := range runningTotalCostPerDate {
		output.WriteString(fmt.Sprintf("%s: %.2f\n", dateStr, cost))
	}
	output.WriteString(fmt.Sprintf("Running Average Cost to Date: %.2f\n", runningAverageCostToDate))
	output.WriteString(fmt.Sprintf("Number of Messages Processed So Far: %d\n", messagesProcessedSoFar))
	output.WriteString("```")

	// Print the message in the terminal
	log.Println(output.String())

	// Send the message to Slack
	slackWebhookURL := "https://hooks.slack.com/services/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"
	payload := map[string]string{"text": output.String()}
	payloadBytes, _ := json.Marshal(payload)
	_, err = http.Post(slackWebhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Fatalf("Error sending message to Slack: %v", err)
	}
}

func fetchData(ctx context.Context, client *spanner.Client) ([]Message, error) {
	query := `
		SELECT
			id,
			date,
			service,
			description,
			cost
		FROM
			jet_tbl
	`

	iter := client.Single().Query(ctx, spanner.NewStatement(query))

	var messages []Message
	for {
		var msg Message
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := row.ToStruct(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func calculateRunningTotalCost(messages []Message) float64 {
	var totalCost float64
	for _, msg := range messages {
		totalCost += msg.Cost
	}
	return totalCost
}

func calculateRunningTotalCostPerDate(messages []Message) map[string]float64 {
	costPerDate := make(map[string]float64)
	for _, msg := range messages {
		if !msg.Date.IsNull() {
			dateStr := msg.Date.String()
			costPerDate[dateStr] += msg.Cost
		}
	}
	return costPerDate
}

func calculateRunningAverageCost(messages []Message) float64 {
	var totalCost float64
	for _, msg := range messages {
		totalCost += msg.Cost
	}
	return totalCost / float64(len(messages))
}
