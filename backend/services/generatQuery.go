package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hupe1980/go-huggingface"
)

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

func GgenerateQuery(prompt string) (string, error) {
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
