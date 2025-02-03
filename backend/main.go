package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/hupe1980/go-huggingface"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

// Request and Response structures
type QueryRequest struct {
	Prompt string `json:"prompt"`
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

// Define table schema structures
type TableSchema struct {
	Name        string         `json:"name"`
	Columns     []ColumnSchema `json:"columns"`
	Description string         `json:"description"`
}

type ColumnSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Nullable    bool   `json:"nullable"`
	Description string `json:"description"`
}

type QueryResponse struct {
	Data       []map[string]interface{} `json:"data"`
	GraphTypes []string                 `json:"graph_types"`
	Error      string                   `json:"error,omitempty"`
}
type ChartConfiguration struct {
	ChartType string        `json:"chartType"`
	XLabel    string        `json:"xLabel"`
	YLabel    string        `json:"yLabel"`
	Labels    []interface{} `json:"labels"`
	Values    []interface{} `json:"values"`
	Title     string        `json:"title"`
	Insights  string        `json:"insights,omitempty"`
}

var databaseSchema = []TableSchema{
	{
		Name:        "departments",
		Description: "Stores department information",
		Columns: []ColumnSchema{
			{Name: "department_id", Type: "serial", Nullable: false, Description: "Primary key"},
			{Name: "name", Type: "varchar(100)", Nullable: false, Description: "Department name"},
			{Name: "location", Type: "varchar(100)", Nullable: false, Description: "Location of the department"},
		},
	},
	{
		Name:        "employees",
		Description: "Stores employee information",
		Columns: []ColumnSchema{
			{Name: "employee_id", Type: "serial", Nullable: false, Description: "Primary key"},
			{Name: "name", Type: "varchar(100)", Nullable: false, Description: "Employee name"},
			{Name: "department_id", Type: "int", Nullable: false, Description: "Reference to departments table"},
			{Name: "hire_date", Type: "date", Nullable: false, Description: "Hire date of the employee"},
			{Name: "salary", Type: "numeric(10,2)", Nullable: false, Description: "Salary of the employee"},
		},
	},
	{
		Name:        "sales",
		Description: "Stores sales records",
		Columns: []ColumnSchema{
			{Name: "sale_id", Type: "serial", Nullable: false, Description: "Primary key"},
			{Name: "employee_id", Type: "int", Nullable: false, Description: "Reference to employees table"},
			{Name: "sale_date", Type: "date", Nullable: false, Description: "Date of the sale"},
			{Name: "amount", Type: "numeric(10,2)", Nullable: false, Description: "Amount of the sale"},
		},
	},
	{
		Name:        "projects",
		Description: "Stores project information",
		Columns: []ColumnSchema{
			{Name: "project_id", Type: "serial", Nullable: false, Description: "Primary key"},
			{Name: "name", Type: "varchar(100)", Nullable: false, Description: "Project name"},
			{Name: "start_date", Type: "date", Nullable: false, Description: "Start date of the project"},
			{Name: "end_date", Type: "date", Nullable: false, Description: "End date of the project"},
			{Name: "budget", Type: "numeric(10,2)", Nullable: false, Description: "Budget for the project"},
		},
	},
}

var dbPool *pgxpool.Pool

func initDB() {
	// Load database connection details from environment variables
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	fmt.Println("Database Host:", dbHost, "Port:", dbPort, "User:", dbUser, "Password:", dbPassword, "DB Name:", dbName)

	// Construct the connection string
	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Create a connection pool
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}

	// Set connection pool settings (optional)
	config.MaxConns = 10 // Maximum number of connections in the pool

	// Create the connection pool
	dbPool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	// Test the connection
	err = dbPool.Ping(context.Background())
	if err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Successfully connected to the database")
}

func validateSQL(query string) bool {
	// Basic validation: prevent dangerous queries
	blacklist := []string{"DROP", "DELETE", "UPDATE", "ALTER", "TRUNCATE"}
	for _, word := range blacklist {
		if containsIgnoreCase(query, word) {
			return false
		}
	}
	return true
}

func containsIgnoreCase(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || containsIgnoreCase(str[1:], substr))
}

func generateQuery(prompt string) (string, error) {
	// Initialize the Hugging Face client
	ic := huggingface.NewInferenceClient(os.Getenv("API_KEY"))

	// Explicitly ask for a single PostgreSQL query
	req := &huggingface.TextGenerationRequest{
		Inputs: fmt.Sprintf("Return ONLY the PostgreSQL query, no explanations or extra text:\n\n%s\n\nPostgreSQL Query:\n", prompt),
		// Model:  os.Getenv("MODEL_ID"),
		Parameters: huggingface.TextGenerationParameters{
			MaxNewTokens:   intPtr(500),     // Allow more tokens for complex queries
			Temperature:    float64Ptr(0.5), // Keep temperature low for accurate SQL
			TopK:           intPtr(50),
			TopP:           float64Ptr(0.9),
			ReturnFullText: boolPtr(false),
		},
	}

	// Generate the query
	res, err := ic.TextGeneration(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("text generation error: %v", err)
	}

	if len(res) == 0 {
		return "", fmt.Errorf("no query generated")
	}

	// Clean up the generated SQL output
	generatedQuery := strings.TrimSpace(res[0].GeneratedText)
	generatedQuery = strings.TrimPrefix(generatedQuery, "```sql")
	generatedQuery = strings.TrimSuffix(generatedQuery, "```")
	generatedQuery = strings.TrimSpace(generatedQuery)

	// Validate the query
	if generatedQuery == "" {
		return "", fmt.Errorf("generated query is empty")
	}

	// Ensure the query starts with a valid SQL keyword
	lowerQuery := strings.ToLower(generatedQuery)
	validPrefixes := []string{"select", "insert", "update", "delete", "with"}
	isValid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(lowerQuery, prefix) {
			isValid = true
			break
		}
	}

	if !isValid {
		return "", fmt.Errorf("generated text does not appear to be a valid PostgreSQL query: %s", generatedQuery)
	}

	return generatedQuery, nil
}

func refineQueryWithSchema(initialQuery string, schema []TableSchema) (string, error) {
	ic := huggingface.NewInferenceClient(os.Getenv("API_KEY"))

	// Create a schema description
	var schemaDesc strings.Builder
	schemaDesc.WriteString("Database Schema:\n")

	for _, table := range schema {
		schemaDesc.WriteString(fmt.Sprintf("- %s: %s\n", table.Name, table.Description))
		schemaDesc.WriteString("  Columns: ")
		for i, col := range table.Columns {
			nullable := ""
			if !col.Nullable {
				nullable = "NOT NULL"
			}
			schemaDesc.WriteString(fmt.Sprintf("%s (%s %s)", col.Name, col.Type, nullable))
			if i < len(table.Columns)-1 {
				schemaDesc.WriteString(", ")
			}
		}
		schemaDesc.WriteString("\n")
	}

	// Create concise query prompt
	prompt := fmt.Sprintf(
		`Generate only the PostgreSQL plain query without any explanation.
		You should follow the Database Schema:
		%s

		Query Requirements:
		- replace the tables and columns names with database schema.
		- no explanation or extra text only query.
		- Should follow the schema for column types and constraints.

		Query:
		%s

		PostgreSQL Query:\n`,
		schemaDesc.String(), initialQuery,
	)

	fmt.Println("Prompt:", prompt)

	// Prepare the text generation request
	req := &huggingface.TextGenerationRequest{
		Inputs: prompt,
		// Model:  os.Getenv("MODEL_ID"),
		Parameters: huggingface.TextGenerationParameters{
			MaxNewTokens:   intPtr(800),     // Increased for complex queries
			Temperature:    float64Ptr(0.4), // Lower for more precise output
			TopK:           intPtr(50),
			TopP:           float64Ptr(0.95),
			ReturnFullText: boolPtr(false),
		},
	}

	// Generate the refined query
	res, err := ic.TextGeneration(context.Background(), req)
	fmt.Println("Generated Response:", res)

	if err != nil {
		return "", fmt.Errorf("query refinement error: %v", err)
	}

	if len(res) == 0 {
		return "", fmt.Errorf("no refined query generated")
	}

	// Clean up the generated query
	refinedQuery := res[0].GeneratedText
	refinedQuery = strings.TrimSpace(refinedQuery)

	// Remove Markdown code blocks (if present)
	refinedQuery = strings.TrimPrefix(refinedQuery, "```sql")
	refinedQuery = strings.TrimSuffix(refinedQuery, "```")
	refinedQuery = strings.TrimSpace(refinedQuery)

	// Remove newline characters and extra spaces
	refinedQuery = strings.ReplaceAll(refinedQuery, "\n", " ")
	refinedQuery = strings.Join(strings.Fields(refinedQuery), " ") // Remove extra spaces

	// Validate the refined query
	if refinedQuery == "" {
		return "", fmt.Errorf("refined query is empty")
	}

	// Basic validation: Ensure the query starts with a valid SQL keyword
	lowerQuery := strings.ToLower(refinedQuery)
	validPrefixes := []string{"select", "insert", "update", "delete", "with"}
	isValid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(lowerQuery, prefix) {
			isValid = true
			break
		}
	}

	if !isValid {
		return "", fmt.Errorf("generated text does not appear to be a valid PostgreSQL query: %s", refinedQuery)
	}

	return refinedQuery, nil
}
func getFinalQueryWithRefinedQuery(refinedQuery string) (string, error) {
	// Initialize the Hugging Face client
	ic := huggingface.NewInferenceClient(os.Getenv("API_KEY"))

	// Create a very explicit prompt for the LLM
	prompt := fmt.Sprintf(
		`Instruction: Generate ONLY a valid PostgreSQL query based on the following context. 
NO ADDITIONAL TEXT. NO EXPLANATION. 
PURE SQL QUERY ONLY:

Context: %s
QUERY:`, refinedQuery,
	)

	// Prepare the text generation request
	req := &huggingface.TextGenerationRequest{
		Inputs: prompt,
		Parameters: huggingface.TextGenerationParameters{
			MaxNewTokens:   intPtr(800),
			Temperature:    float64Ptr(0.1), // Very low temperature for precision
			TopK:           intPtr(10),
			TopP:           float64Ptr(0.9),
			ReturnFullText: boolPtr(false),
		},
	}

	// Generate the response
	res, err := ic.TextGeneration(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("LLM query generation error: %v", err)
	}

	if len(res) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	// Extract the query with multiple cleaning strategies
	generatedText := res[0].GeneratedText
	var finalQuery string

	// Strategy 1: Direct extraction
	finalQuery = extractSQLQuery(generatedText)

	// If first strategy fails, try alternative strategies
	if finalQuery == "" {
		// Strategy 2: Remove everything before SELECT/INSERT/UPDATE/DELETE/WITH
		matches := regexp.MustCompile(`(SELECT|INSERT|UPDATE|DELETE|WITH).*`).FindStringSubmatch(generatedText)
		if len(matches) > 1 {
			finalQuery = matches[0]
		}
	}

	// Final cleaning
	finalQuery = strings.TrimSpace(finalQuery)
	finalQuery = strings.Trim(finalQuery, "`;\"'")
	finalQuery = regexp.MustCompile(`^.*?(\bSELECT\b|\bINSERT\b|\bUPDATE\b|\bDELETE\b|\bWITH\b)`).ReplaceAllString(finalQuery, "$1")

	// Validate the query
	if finalQuery == "" {
		return "", fmt.Errorf("could not extract a valid query from the LLM response")
	}

	// Strict validation of query start
	lowerQuery := strings.ToLower(finalQuery)
	validPrefixes := []string{"select", "insert", "update", "delete", "with"}
	isValid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(lowerQuery, prefix) {
			isValid = true
			break
		}
	}

	if !isValid {
		return "", fmt.Errorf("generated text does not appear to be a valid PostgreSQL query: %s", finalQuery)
	}

	return finalQuery, nil
}

// Helper function to extract SQL query
func extractSQLQuery(text string) string {
	// Regular expressions to extract SQL query
	patterns := []string{
		`\b(SELECT|INSERT|UPDATE|DELETE|WITH)\s.*?;`,
		`\b(SELECT|INSERT|UPDATE|DELETE|WITH)\s[^;]*`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

func handleGenerateQuery(w http.ResponseWriter, r *http.Request) {
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
	initialQuery, err := generateQuery(req.Prompt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(QueryResponse{
			Error: fmt.Sprintf("initial query generation error: %v", err),
		})
		return
	}

	// Step 2: Refine query
	refinedQuery, err := refineQueryWithSchema(initialQuery, databaseSchema)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"initial_query": "%s", "error": "query refinement error: %v"}`, initialQuery, err), http.StatusInternalServerError)
		return
	}

	finalQuery, err := getFinalQueryWithRefinedQuery(refinedQuery)

	fmt.Println("Final Query:", finalQuery)

	if err != nil {
		http.Error(w, fmt.Sprintf(`{"refined_query": "%s", "error": "final query generation error: %v"}`, refinedQuery, err), http.StatusInternalServerError)
		return
	}
	// Step 3: Validate query before execution
	if !validateSQL(finalQuery) {
		http.Error(w, fmt.Sprintf(`{"initial_query": "%s", "final_query": "%s", "error": "query contains forbidden operations"}`, initialQuery, finalQuery), http.StatusBadRequest)
		return
	}

	// Debug: Check if dbPool is nil
	if dbPool == nil {
		log.Println("dbPool is nil. Database connection not initialized.")
		http.Error(w, `{"error": "Database connection not initialized"}`, http.StatusInternalServerError)
		return
	}

	// Step 4: Execute the query
	rows, err := dbPool.Query(context.Background(), finalQuery)
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

	chartConfig, err := generateChartConfigurations(results, "Show salary distribution")
	if err != nil {
		log.Fatal(err)
	}

	// Safely access fields
	fmt.Println(chartConfig.ChartType)
	fmt.Println(chartConfig.XLabel)

	// Convert to Chart.js format
	chartJSConfig, options := parseChartConfigToChartJS(chartConfig)
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

// Optional helper function for parsing
func parseChartConfigToChartJS(chartConfig *ChartConfiguration) (map[string]interface{}, map[string]interface{}) {
	chartJSConfig := map[string]interface{}{
		"labels": chartConfig.Labels,
		"datasets": []map[string]interface{}{
			{
				"label":           chartConfig.YLabel,
				"data":            chartConfig.Values,
				"backgroundColor": "rgba(59, 130, 246, 0.5)",
			},
		},
	}

	options := map[string]interface{}{
		"responsive": true,
		"plugins": map[string]interface{}{
			"title": map[string]interface{}{
				"display": true,
				"text":    chartConfig.Title,
			},
		},
		"scales": map[string]interface{}{
			"x": map[string]interface{}{
				"title": map[string]interface{}{
					"display": true,
					"text":    chartConfig.XLabel,
				},
			},
			"y": map[string]interface{}{
				"title": map[string]interface{}{
					"display": true,
					"text":    chartConfig.YLabel,
				},
			},
		},
	}

	return chartJSConfig, options
}

func generateChartConfigurations(results []map[string]interface{}, prompt string) (*ChartConfiguration, error) {
	// Initialize the Hugging Face client
	ic := huggingface.NewInferenceClient(os.Getenv("API_KEY"))

	// Prepare the input data as a JSON string
	dataJSON, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input data: %v", err)
	}

	// Create a detailed prompt for the LLM
	fullPrompt := fmt.Sprintf(`
You are a data visualization expert. Given the following JSON data and user prompt, generate a comprehensive chart configuration.

User Prompt: %s

Input Data (JSON): %s

Guidelines for Chart Configuration:
1. Analyze the data structure and content
2. Choose the most appropriate chart type
3. Select meaningful x and y axis data
4. Create descriptive labels
5. Provide insights about the data visualization

Return a JSON configuration with these fields:
- chartType (bar/line/pie/scatter/radar)
- xLabel (x-axis label)
- yLabel (y-axis label)
- labels (x-axis categories)
- values (y-axis numeric values)
- title (chart title)
- insights (optional explanation)

IMPORTANT: Return ONLY a valid JSON matching this structure.`, prompt, string(dataJSON))

	// Prepare the text generation request
	req := &huggingface.TextGenerationRequest{
		Inputs: fullPrompt,
		Parameters: huggingface.TextGenerationParameters{
			MaxNewTokens:   intPtr(1000),
			Temperature:    float64Ptr(0.3),
			TopK:           intPtr(10),
			TopP:           float64Ptr(0.9),
			ReturnFullText: boolPtr(false),
		},
	}

	// Generate the response
	res, err := ic.TextGeneration(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("LLM chart configuration generation error: %v", err)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Extract and clean the JSON response
	generatedText := res[0].GeneratedText
	generatedText = strings.TrimSpace(generatedText)

	// Remove any text before the first '{' and after the last '}'
	jsonStart := strings.Index(generatedText, "{")
	jsonEnd := strings.LastIndex(generatedText, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("could not extract valid JSON from LLM response")
	}
	generatedText = generatedText[jsonStart : jsonEnd+1]

	// Parse the JSON into our structured type
	var chartConfig ChartConfiguration
	err = json.Unmarshal([]byte(generatedText), &chartConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response JSON: %v", err)
	}

	// Validate the configuration
	if chartConfig.ChartType == "" || chartConfig.XLabel == "" || chartConfig.YLabel == "" {
		return nil, fmt.Errorf("invalid chart configuration: missing required fields")
	}

	return &chartConfig, nil
}

// Helper function to safely convert interface{} to []interface{}
func convertToSlice(v interface{}) ([]interface{}, error) {
	switch typed := v.(type) {
	case []interface{}:
		return typed, nil
	case []map[string]interface{}:
		result := make([]interface{}, len(typed))
		for i, item := range typed {
			result[i] = item
		}
		return result, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to slice", v)
	}
}

func main() {
	// Load .env file
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Check for API key
	if os.Getenv("API_KEY") == "" {
		log.Fatal("API_KEY environment variable is required")
	}

	// Initialize DB connection
	initDB()

	mux := http.NewServeMux()

	// Register your handlers
	mux.HandleFunc("/generate-query", handleGenerateQuery)

	// Create a CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // All origins
		AllowedMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders: []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization"},
		Debug:          true, // Enable debugging for testing, remove in production
	})

	// Create handler with CORS
	handler := c.Handler(mux)
	// Start server
	PORT := os.Getenv("PORT")
	log.Printf("Server starting on port %s", PORT)
	if err := http.ListenAndServe(PORT, handler); err != nil {
		log.Fatal(err)
	}
}
