package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

// Comprehensive predefined schema
var databaseSchema = []TableSchema{
	{
		Name:        "users",
		Description: "Stores user account information",
		Columns: []ColumnSchema{
			{Name: "id", Type: "bigserial", Nullable: false, Description: "Primary key"},
			{Name: "email", Type: "varchar(255)", Nullable: false, Description: "User's email address"},
			{Name: "username", Type: "varchar(50)", Nullable: false, Description: "Username for login"},
			{Name: "password_hash", Type: "varchar(255)", Nullable: false, Description: "Hashed password"},
			{Name: "full_name", Type: "varchar(100)", Nullable: false, Description: "User's full name"},
			{Name: "created_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of account creation"},
			{Name: "updated_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of last update"},
			{Name: "last_login", Type: "bigint", Nullable: true, Description: "Unix timestamp of last login"},
			{Name: "is_active", Type: "boolean", Nullable: false, Description: "Account status"},
			{Name: "role", Type: "varchar(20)", Nullable: false, Description: "User role (admin, user, etc.)"},
		},
	},
	{
		Name:        "products",
		Description: "Product catalog",
		Columns: []ColumnSchema{
			{Name: "id", Type: "bigserial", Nullable: false, Description: "Primary key"},
			{Name: "name", Type: "varchar(100)", Nullable: false, Description: "Product name"},
			{Name: "description", Type: "text", Nullable: true, Description: "Product description"},
			{Name: "price", Type: "decimal(10,2)", Nullable: false, Description: "Current price"},
			{Name: "stock", Type: "integer", Nullable: false, Description: "Available stock"},
			{Name: "category_id", Type: "bigint", Nullable: false, Description: "Reference to categories table"},
			{Name: "created_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of creation"},
			{Name: "updated_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of last update"},
			{Name: "is_active", Type: "boolean", Nullable: false, Description: "Product status"},
		},
	},
	{
		Name:        "categories",
		Description: "Product categories",
		Columns: []ColumnSchema{
			{Name: "id", Type: "bigserial", Nullable: false, Description: "Primary key"},
			{Name: "name", Type: "varchar(50)", Nullable: false, Description: "Category name"},
			{Name: "description", Type: "text", Nullable: true, Description: "Category description"},
			{Name: "parent_id", Type: "bigint", Nullable: true, Description: "Self-reference for hierarchical categories"},
			{Name: "created_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of creation"},
			{Name: "is_active", Type: "boolean", Nullable: false, Description: "Category status"},
		},
	},
	{
		Name:        "orders",
		Description: "Customer orders",
		Columns: []ColumnSchema{
			{Name: "id", Type: "bigserial", Nullable: false, Description: "Primary key"},
			{Name: "user_id", Type: "bigint", Nullable: false, Description: "Reference to users table"},
			{Name: "status", Type: "varchar(20)", Nullable: false, Description: "Order status (pending, completed, etc.)"},
			{Name: "total_amount", Type: "decimal(12,2)", Nullable: false, Description: "Total order amount"},
			{Name: "created_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of order creation"},
			{Name: "updated_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of last update"},
			{Name: "payment_status", Type: "varchar(20)", Nullable: false, Description: "Payment status"},
		},
	},
	{
		Name:        "order_items",
		Description: "Individual items within orders",
		Columns: []ColumnSchema{
			{Name: "id", Type: "bigserial", Nullable: false, Description: "Primary key"},
			{Name: "order_id", Type: "bigint", Nullable: false, Description: "Reference to orders table"},
			{Name: "product_id", Type: "bigint", Nullable: false, Description: "Reference to products table"},
			{Name: "quantity", Type: "integer", Nullable: false, Description: "Quantity ordered"},
			{Name: "unit_price", Type: "decimal(10,2)", Nullable: false, Description: "Price at time of order"},
			{Name: "created_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of creation"},
		},
	},
	{
		Name:        "reviews",
		Description: "Product reviews by users",
		Columns: []ColumnSchema{
			{Name: "id", Type: "bigserial", Nullable: false, Description: "Primary key"},
			{Name: "product_id", Type: "bigint", Nullable: false, Description: "Reference to products table"},
			{Name: "user_id", Type: "bigint", Nullable: false, Description: "Reference to users table"},
			{Name: "rating", Type: "integer", Nullable: false, Description: "Rating (1-5)"},
			{Name: "comment", Type: "text", Nullable: true, Description: "Review comment"},
			{Name: "created_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of creation"},
			{Name: "updated_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of last update"},
			{Name: "is_verified", Type: "boolean", Nullable: false, Description: "Verified purchase review"},
		},
	},
	{
		Name:        "inventory_log",
		Description: "Product inventory changes log",
		Columns: []ColumnSchema{
			{Name: "id", Type: "bigserial", Nullable: false, Description: "Primary key"},
			{Name: "product_id", Type: "bigint", Nullable: false, Description: "Reference to products table"},
			{Name: "quantity_change", Type: "integer", Nullable: false, Description: "Change in quantity (positive/negative)"},
			{Name: "type", Type: "varchar(20)", Nullable: false, Description: "Type of change (restock, order, adjustment)"},
			{Name: "reference_id", Type: "bigint", Nullable: true, Description: "Reference to related record (order_id etc)"},
			{Name: "created_at", Type: "bigint", Nullable: false, Description: "Unix timestamp of change"},
			{Name: "created_by", Type: "bigint", Nullable: false, Description: "Reference to users table"},
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

func analyzeDataForGraphs(results []map[string]interface{}) ([]string, error) {
	// Initialize the Hugging Face client
	ic := huggingface.NewInferenceClient(os.Getenv("API_KEY"))

	// Convert results to a JSON string for the LLM prompt
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %v", err)
	}

	// Create a prompt for the LLM
	prompt := fmt.Sprintf(
		`Analyze the following data and suggest suitable graph types (e.g., Bar Chart, Line Chart, Pie Chart, Scatter Plot, Generic Table). 
		Return ONLY the graph types as a comma-separated list, no explanations or extra text.

		Data:
		%s

		Suggested Graph Types:`, resultsJSON,
	)

	// Prepare the text generation request
	req := &huggingface.TextGenerationRequest{
		Inputs: prompt,
		Parameters: huggingface.TextGenerationParameters{
			MaxNewTokens:   intPtr(100),     // Limit tokens for concise output
			Temperature:    float64Ptr(0.4), // Lower for more precise output
			TopK:           intPtr(50),
			TopP:           float64Ptr(0.95),
			ReturnFullText: boolPtr(false),
		},
	}

	// Generate the response
	res, err := ic.TextGeneration(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis error: %v", err)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Clean up the response
	graphTypes := strings.TrimSpace(res[0].GeneratedText)
	graphTypes = strings.TrimPrefix(graphTypes, "```")
	graphTypes = strings.TrimSuffix(graphTypes, "```")
	graphTypes = strings.TrimSpace(graphTypes)

	// Split the comma-separated list into a slice
	graphTypeList := strings.Split(graphTypes, ",")
	for i := range graphTypeList {
		graphTypeList[i] = strings.TrimSpace(graphTypeList[i])
	}

	return graphTypeList, nil
}

func generateQuery(prompt string) (string, error) {
	// Initialize the Hugging Face client
	ic := huggingface.NewInferenceClient(os.Getenv("API_KEY"))

	// Explicitly ask for a single PostgreSQL query
	req := &huggingface.TextGenerationRequest{
		Inputs: fmt.Sprintf("Return ONLY the PostgreSQL query, no explanations or extra text:\n\n%s\n\nPostgreSQL Query:\n", prompt),
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
		- Ensure proper table relationships and joins.
		- Handle timestamps as bigint (Unix timestamp).

		Query:
		%s

		PostgreSQL Query:\n`,
		schemaDesc.String(), initialQuery,
	)

	fmt.Println("Prompt:", prompt)

	// Prepare the text generation request
	req := &huggingface.TextGenerationRequest{
		Inputs: prompt,
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

	// Create a prompt for the LLM
	prompt := fmt.Sprintf(
		`Ensure the query is syntactically correct and optimized for execution without any explanation and extra words.
		Fix syntax errors in my PostgreSQL query where double quotes (\") are incorrectly escaped.

		Refined Query:
		%s
	
		PostgreSQL Query:`, refinedQuery,
	)

	// Prepare the text generation request
	req := &huggingface.TextGenerationRequest{
		Inputs: prompt,
		Parameters: huggingface.TextGenerationParameters{
			MaxNewTokens:   intPtr(500),     // Allow more tokens for complex queries
			Temperature:    float64Ptr(0.5), // Keep temperature low for accurate SQL
			TopK:           intPtr(50),
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

	// Clean up the response
	finalQuery := strings.TrimSpace(res[0].GeneratedText)
	finalQuery = strings.TrimPrefix(finalQuery, "```sql")
	finalQuery = strings.TrimSuffix(finalQuery, "```")
	finalQuery = strings.TrimSpace(finalQuery)

	// Validate the query
	if finalQuery == "" {
		return "", fmt.Errorf("generated query is empty")
	}

	// Ensure the query starts with a valid SQL keyword
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

	// Check for errors after iterating over rows
	if err := rows.Err(); err != nil {
		log.Printf("Error after iterating rows: %v", err)
		http.Error(w, `{"error": "Error after iterating rows"}`, http.StatusInternalServerError)
		return
	}

	// Step 5: Analyze data for suitable graph types
	graphTypes, err := analyzeDataForGraphs(results)
	if err != nil {
		log.Printf("Failed to analyze data for graphs: %v", err)
		http.Error(w, `{"error": "Failed to analyze data for graphs"}`, http.StatusInternalServerError)
		return
	}

	// Construct response object
	response := map[string]interface{}{
		"data":       results,
		"graphTypes": graphTypes,
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
