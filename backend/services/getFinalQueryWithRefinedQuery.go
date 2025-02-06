package services

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hupe1980/go-huggingface"
)

func GetFinalQueryWithRefinedQuery(refinedQuery string) (string, error) {
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
