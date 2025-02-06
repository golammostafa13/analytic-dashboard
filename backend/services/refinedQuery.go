package services

import (
	"backend/database"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hupe1980/go-huggingface"
)

func RefineQueryWithSchema(initialQuery string, schema []database.TableSchema) (string, error) {
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