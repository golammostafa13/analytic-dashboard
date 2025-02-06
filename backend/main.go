package main

import (
	"backend/database"
	"backend/query"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

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
	database.InitDB()

	mux := http.NewServeMux()

	// Register your handlers
	mux.HandleFunc("/generate-query", query.HandleGenerateQuery)

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
