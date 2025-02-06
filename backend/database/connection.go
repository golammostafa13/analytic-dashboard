package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DbPool *pgxpool.Pool

func InitDB() {
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
	DbPool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	// Test the connection
	err = DbPool.Ping(context.Background())
	if err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Successfully connected to the database")
}
