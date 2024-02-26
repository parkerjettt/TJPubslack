package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/parkerjettt/tjfunc"
	"google.golang.org/api/iterator"
)

type CostDataResponse struct {
	ChartData ChartData `json:"chart_data"`
	CostData  CostData  `json:"cost_data"`
}

type CostData struct {
	RunningTotalCost        float64      `json:"running_total_cost"`
	RunningTotalCostPerDate []CostRecord `json:"running_total_cost_per_date"`
	RunningAverageCost      float64      `json:"running_average_cost"`
	NumMessagesProcessed    int64        `json:"num_messages_processed"`
}

type CostRecord struct {
	Date   spanner.NullDate `json:"date"`
	Cost   float64          `json:"cost"`
	Amount int64            `json:"amount"`
}

type ChartData struct {
	Labels   []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

type Dataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BorderColor     string    `json:"borderColor"`
	BackgroundColor string    `json:"backgroundColor"`
	BorderWidth     int       `json:"borderWidth"`
	Fill            bool      `json:"fill"`
}

func main() {
	http.HandleFunc("/data", handleDataRequest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleDataRequest(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Fetch data from your database or API
	ctx := context.Background()

	projectID := "alphaus-live"
	instanceID := "intern2024ft"
	databaseID := "default"

	client, err := spanner.NewClient(ctx, "projects/"+projectID+"/instances/"+instanceID+"/databases/"+databaseID)
	if err != nil {
		http.Error(w, "Failed to create Spanner client", http.StatusInternalServerError)
		log.Printf("Failed to create Spanner client: %v", err)
		return
	}
	defer client.Close()

	// Fetching data for chart
	costToDateRecords := getRunningTotalCostToDatechart(ctx, client)
	costPerDateRecords := tjfunc.GetRunningTotalCostPerDate(ctx, client)

	// Populate the labels array with date values
	var labels []string
	for _, record := range costToDateRecords {
		labels = append(labels, record.Date.String())
	}

	// Transform the fetched data into the format expected by Chart.js
	var costToDateValues, costPerDateValues []float64
	for _, record := range costToDateRecords {
		costToDateValues = append(costToDateValues, record.Cost)
	}
	for _, record := range costPerDateRecords {
		costPerDateValues = append(costPerDateValues, record.Cost)
	}

	// Creating the chart data struct
	chartData := ChartData{
		Labels: labels,
		Datasets: []Dataset{
			{
				Label:           "Total Cost to Date",
				Data:            costToDateValues,
				BorderColor:     "rgba(255, 99, 132, 1)",
				BackgroundColor: "rgba(255, 99, 132, 0.2)",
				BorderWidth:     2,
				Fill:            true,
			},
			{
				Label:           "Total Cost per Date",
				Data:            costPerDateValues,
				BorderColor:     "rgba(54, 162, 235, 1)",
				BackgroundColor: "rgba(54, 162, 235, 0.2)",
				BorderWidth:     2,
				Fill:            true,
			},
		},
	}

	// Fetching data for cost
	totalCostToDate := getRunningTotalCostToDate(ctx, client)
	totalCostPerDate := getRunningTotalCostPerDate(ctx, client)
	averageCostToDate := tjfunc.GetRunningAverageCostToDate(ctx, client)
	numMessagesProcessed := tjfunc.GetNumMessagesProcessed(ctx, client)

	// Calculate running total cost
	var runningTotalCost float64
	for _, record := range totalCostToDate {
		runningTotalCost += record.Cost
	}

	costData := CostData{
		RunningTotalCost:        runningTotalCost,
		RunningTotalCostPerDate: totalCostPerDate,
		RunningAverageCost:      averageCostToDate,
		NumMessagesProcessed:    numMessagesProcessed,
	}

	// Create a combined response object
	costDataResponse := CostDataResponse{
		ChartData: chartData,
		CostData:  costData,
	}

	// Encode the combined response object as JSON and send it in the response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(costDataResponse)
	if err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		log.Printf("Failed to encode JSON response: %v", err)
		return
	}
}
func getRunningTotalCostToDatechart(ctx context.Context, client *spanner.Client) []CostRecord {
	stmt := spanner.Statement{
		SQL: `SELECT date, SUM(cost) AS total_cost FROM jet_tbl GROUP BY date ORDER BY date`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var runningTotalCostPerDate []CostRecord
	var runningTotalCost float64

	for {
		var record CostRecord
		var date spanner.NullDate
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Error fetching results: %v", err)
		}
		if err := row.Columns(&date, &record.Cost); err != nil {
			log.Fatalf("Error reading row: %v", err)
		}

		runningTotalCost += record.Cost
		record.Date = date
		record.Cost = runningTotalCost
		runningTotalCostPerDate = append(runningTotalCostPerDate, record)
	}

	return runningTotalCostPerDate
}

func getRunningTotalCostToDate(ctx context.Context, client *spanner.Client) []CostRecord {
	stmt := spanner.Statement{
		SQL: `SELECT date, SUM(cost) AS total_cost FROM jet_tbl GROUP BY date ORDER BY date`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var runningTotalCostPerDate []CostRecord
	for {
		var record CostRecord
		var date spanner.NullDate
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Error fetching results: %v", err)
		}
		if err := row.Columns(&date, &record.Cost); err != nil {
			log.Fatalf("Error reading row: %v", err)
		}
		record.Date = date
		runningTotalCostPerDate = append(runningTotalCostPerDate, record)
	}

	return runningTotalCostPerDate
}

func getRunningTotalCostPerDate(ctx context.Context, client *spanner.Client) []CostRecord {
	stmt := spanner.Statement{
		SQL: `SELECT date, SUM(cost) AS total_cost FROM jet_tbl GROUP BY date ORDER BY date`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var runningTotalCostPerDate []CostRecord
	for {
		var record CostRecord
		var date spanner.NullDate
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Error fetching results: %v", err)
		}
		if err := row.Columns(&date, &record.Cost); err != nil {
			log.Fatalf("Error reading row: %v", err)
		}
		record.Date = date
		runningTotalCostPerDate = append(runningTotalCostPerDate, record)
	}

	return runningTotalCostPerDate
}
