package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// APIModel represents a model from the OpenAI-compatible /models endpoint
type APIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse represents the response from /models endpoint
type ModelsResponse struct {
	Object string     `json:"object"`
	Data   []APIModel `json:"data"`
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available LLM models from API",
	Long: `Fetch and list available LLM models from the configured API endpoint.

This command queries the /models endpoint of your LLM provider to show
all available models. Requires LLM_API_KEY and LLM_BASE_URL to be set.

To use a specific model, set the environment variables:
  LLM_MODEL=<model-id>         # For text extraction
  LLM_VISION_MODEL=<model-id>  # For vision/image extraction

Or use CLI flags:
  --llm-model <model-id>
  --llm-vision-model <model-id>`,
	RunE: runModels,
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}

func runModels(cmd *cobra.Command, args []string) error {
	// Show current configuration first
	fmt.Println("Current Configuration:")
	fmt.Println("----------------------")

	baseURL := llmBaseURL
	if baseURL == "" {
		baseURL = os.Getenv("LLM_BASE_URL")
	}
	if baseURL == "" {
		fmt.Println("  LLM_BASE_URL:     (not set)")
		fmt.Println()
		fmt.Println("⚠️  LLM_BASE_URL is required. Set it via environment variable or --llm-base-url flag.")
		return nil
	}

	currentModel := llmModel
	if currentModel == "" {
		currentModel = os.Getenv("LLM_MODEL")
	}
	if currentModel == "" {
		currentModel = "(not set)"
	}

	currentVisionModel := llmVisionModel
	if currentVisionModel == "" {
		currentVisionModel = os.Getenv("LLM_VISION_MODEL")
	}
	if currentVisionModel == "" {
		currentVisionModel = "(not set)"
	}

	currentAPIKey := apiKey
	if currentAPIKey == "" {
		currentAPIKey = os.Getenv("LLM_API_KEY")
	}
	apiKeyStatus := "Not set"
	if currentAPIKey != "" {
		if len(currentAPIKey) > 8 {
			apiKeyStatus = "Set (" + currentAPIKey[:8] + "...)"
		} else {
			apiKeyStatus = "Set"
		}
	}

	fmt.Printf("  LLM_BASE_URL:     %s\n", baseURL)
	fmt.Printf("  LLM_MODEL:        %s\n", currentModel)
	fmt.Printf("  LLM_VISION_MODEL: %s\n", currentVisionModel)
	fmt.Printf("  LLM_API_KEY:      %s\n", apiKeyStatus)
	fmt.Println()

	// Check if API key is set
	if currentAPIKey == "" {
		fmt.Println()
		fmt.Println("⚠️  LLM_API_KEY is required. Set it via environment variable or --api-key flag.")
		return nil
	}

	// Fetch models from API
	fmt.Printf("Fetching models from %s/models...\n", baseURL)
	fmt.Println()

	models, err := fetchModels(baseURL, currentAPIKey)
	if err != nil {
		fmt.Printf("⚠️  Could not fetch models: %v\n", err)
		fmt.Println()
		fmt.Println("Tip: Your API provider may not support the /models endpoint.")
		fmt.Println("     You can still use models by setting LLM_MODEL and LLM_VISION_MODEL directly.")
		return nil
	}

	if len(models) == 0 {
		fmt.Println("No models returned from API.")
		return nil
	}

	// Sort models by ID
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	// Display models
	fmt.Printf("Available Models (%d):\n", len(models))
	fmt.Println("=====================")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "MODEL ID\tOWNER\tCREATED")
	fmt.Fprintln(w, "--------\t-----\t-------")

	for _, m := range models {
		created := ""
		if m.Created > 0 {
			created = time.Unix(m.Created, 0).Format("2006-01-02")
		}
		owner := m.OwnedBy
		if owner == "" {
			owner = inferProvider(m.ID)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", m.ID, owner, created)
	}
	w.Flush()

	return nil
}

func fetchModels(baseURL, apiKey string) ([]APIModel, error) {
	// Build request URL
	url := strings.TrimSuffix(baseURL, "/") + "/models"

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for non-200 status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Check if response is JSON
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") && !strings.HasPrefix(string(body), "{") && !strings.HasPrefix(string(body), "[") {
		return nil, fmt.Errorf("API returned non-JSON response (Content-Type: %s)", contentType)
	}

	// Parse response
	var modelsResp ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		// Try parsing as array directly (some APIs return array instead of object)
		var models []APIModel
		if err2 := json.Unmarshal(body, &models); err2 != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return models, nil
	}

	return modelsResp.Data, nil
}

// inferProvider tries to infer the provider from model ID
func inferProvider(modelID string) string {
	modelID = strings.ToLower(modelID)

	if strings.Contains(modelID, "claude") || strings.Contains(modelID, "anthropic") {
		return "anthropic"
	}
	if strings.Contains(modelID, "gpt") || strings.Contains(modelID, "openai") || strings.Contains(modelID, "o1") || strings.Contains(modelID, "davinci") {
		return "openai"
	}
	if strings.Contains(modelID, "gemini") || strings.Contains(modelID, "google") || strings.Contains(modelID, "palm") {
		return "google"
	}
	if strings.Contains(modelID, "llama") || strings.Contains(modelID, "meta") {
		return "meta"
	}
	if strings.Contains(modelID, "mistral") || strings.Contains(modelID, "mixtral") {
		return "mistral"
	}
	if strings.Contains(modelID, "qwen") {
		return "alibaba"
	}
	if strings.Contains(modelID, "deepseek") {
		return "deepseek"
	}

	return "-"
}
