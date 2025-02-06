package query

import (
	"backend/database"
	"backend/services"
	"backend/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type QueryRequest struct {
	Prompt string `json:"prompt"`
}

type QueryResponse struct {
	Data       []map[string]interface{} `json:"data"`
	GraphTypes []string                 `json:"graph_types"`
	Error      string                   `json:"error,omitempty"`
}

func HandleGenerateQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(QueryResponse{
			Error: "only POST method is allowed",
		})
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(QueryResponse{
			Error: "invalid request body",
		})
		return
	}

	if req.Prompt == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(QueryResponse{
			Error: "prompt is required",
		})
		return
	}

	// Step 1: Generate initial queries
	initialQuery, err := services.GgenerateQuery(req.Prompt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Error: fmt.Sprintf("initial query generation error: %v", err),
		})
		return
	}

	// Step 2: Refine query
	refinedQuery, err := services.RefineQueryWithSchema(initialQuery, database.DatabaseSchema)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"initial_query": "%s", "error": "query refinement error: %v"}`, initialQuery, err), http.StatusInternalServerError)
		return
	}

	finalQuery, err := services.GetFinalQueryWithRefinedQuery(refinedQuery)

	fmt.Println("Final Query:", finalQuery)

	if err != nil {
		http.Error(w, fmt.Sprintf(`{"refined_query": "%s", "error": "final query generation error: %v"}`, refinedQuery, err), http.StatusInternalServerError)
		return
	}
	// Step 3: Validate query before execution
	if !utils.ValidateSQL(finalQuery) {
		http.Error(w, fmt.Sprintf(`{"initial_query": "%s", "final_query": "%s", "error": "query contains forbidden operations"}`, initialQuery, finalQuery), http.StatusBadRequest)
		return
	}

	// Debug: Check if dbPool is nil
	if database.DbPool == nil {
		log.Println("dbPool is nil. Database connection not initialized.")
		http.Error(w, `{"error": "Database connection not initialized"}`, http.StatusInternalServerError)
		return
	}

	// Step 4: Execute the query
	rows, err := database.DbPool.Query(context.Background(), finalQuery)
	if err != nil {
		log.Printf("Query execution failed: %v", err)
		http.Error(w, `{"error": "Error executing query"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Fetch column names and metadata
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name) // Convert pgx.Identifier to string
	}

	// Prepare a slice to store results
	var results []map[string]interface{}

	// Iterate over rows
	for rows.Next() {
		// Create a slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		// Point each element in valuePtrs to the corresponding element in values
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into valuePtrs
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("Failed to scan row: %v", err)
			http.Error(w, `{"error": "Failed to scan row"}`, http.StatusInternalServerError)
			return
		}

		// Create a map to store the row data
		rowData := make(map[string]interface{})

		// Populate the map with column names and values
		for i, colName := range columns {
			var v interface{}
			rawValue := values[i]

			// Handle []byte (e.g., for text-based columns)
			if b, ok := rawValue.([]byte); ok {
				v = string(b) // Convert []byte to string
			} else {
				v = rawValue // Use the value as-is
			}

			rowData[colName] = v
		}

		// Append the row data to the results slice
		results = append(results, rowData)
	}

	chartConfig, err := services.GenerateChartConfigurations(results, "Show salary distribution")
	if err != nil {
		log.Fatal(err)
	}

	// Safely access fields
	fmt.Println(chartConfig.ChartType)
	fmt.Println(chartConfig.XLabel)

	// Convert to Chart.js format
	chartJSConfig, options := utils.ParseChartConfigToChartJS(chartConfig)
	// Now you can use chartConfig directly with type safety
	fmt.Println("Chart Type:", chartConfig.ChartType)
	fmt.Println("Chart.js Config:", chartJSConfig)
	// Check for errors after iterating over rows
	if err := rows.Err(); err != nil {
		log.Printf("Error after iterating rows: %v", err)
		http.Error(w, `{"error": "Error after iterating rows"}`, http.StatusInternalServerError)
		return
	}

	// Step 5: Analyze data for suitable graph types
	graphTypes, err := []string{"Bar Chart", "Line Chart", "Pie Chart", "Area Chart", "Radar Chart", "Table"}, nil

	// Construct response object
	response := map[string]interface{}{
		"data":          results,
		"graphTypes":    graphTypes,
		"chartJSConfig": chartJSConfig,
		"options":       options,
	}

	// Step 6: Return response with results and suggested graph types
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode results to JSON: %v", err)
		http.Error(w, `{"error": "Failed to encode results to JSON"}`, http.StatusInternalServerError)
	}

}
