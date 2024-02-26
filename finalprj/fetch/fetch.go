package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/parkerjettt/tjfunc"
	"google.golang.org/api/option"
)

type CostRecord struct {
	Date   spanner.NullDate
	Cost   float64
	Amount int64
}

func main() {
	projectId := "alphaus-live"

	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/"+projectId+"/instances/intern2024ft/databases/default", option.WithCredentialsFile(`C:\Users\Jet Parks\Internship\intern202401p2.json`))
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer client.Close()
	tjfunc.SendSlackMessage(ctx, client, "https://hooks.slack.com/services/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")
	// Use a ticker to send Slack messages every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go tjfunc.SendSlackMessage(ctx, client, "https://hooks.slack.com/services/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")
		}
	}
}
