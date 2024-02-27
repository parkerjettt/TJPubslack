package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	tjfunc "github.com/parkerjettt/TJPubslack"

	"cloud.google.com/go/spanner"
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
	costToDateRecords := tjfunc.GetRunningTotalCostRecords(ctx, client, true)
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
	totalCostToDate := tjfunc.GetRunningTotalCostRecords(ctx, client, false)
	totalCostPerDate := tjfunc.GetRunningTotalCostPerDateWeb(ctx, client)
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
