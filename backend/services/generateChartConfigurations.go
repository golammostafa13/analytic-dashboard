package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hupe1980/go-huggingface"
)


type ChartConfiguration struct {
	ChartType string        `json:"chartType"`
	XLabel    string        `json:"xLabel"`
	YLabel    string        `json:"yLabel"`
	Labels    []interface{} `json:"labels"`
	Values    []interface{} `json:"values"`
	Title     string        `json:"title"`
	Insights  string        `json:"insights,omitempty"`
}

func GenerateChartConfigurations(results []map[string]interface{}, prompt string) (*ChartConfiguration, error) {
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