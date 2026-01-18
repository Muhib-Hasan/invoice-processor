package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

const (
	DefaultBaseURL = "https://openrouter.ai/api/v1"
	DefaultTimeout = 120 * time.Second
)

// Default models for different tasks
const (
	ModelClaude35Sonnet = "anthropic/claude-3.5-sonnet"
	ModelClaude3Haiku   = "anthropic/claude-3-haiku"
	ModelGPT4oMini      = "openai/gpt-4o-mini"
	ModelGPT4o          = "openai/gpt-4o"
	ModelGeminiFlash    = "google/gemini-flash-1.5"
)

// Client handles communication with OpenAI-compatible APIs
type Client struct {
	client       openai.Client
	visionClient openai.Client // Separate client for vision requests with special headers
	defaultModel string
}

// visionHeaderTransport wraps an http.RoundTripper to add vision-specific headers
type visionHeaderTransport struct {
	base http.RoundTripper
}

func (t *visionHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Copilot-Vision-Request", "true")
	if t.base != nil {
		return t.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// ClientOption configures the client
type ClientOption func(*clientConfig)

type clientConfig struct {
	baseURL      string
	timeout      time.Duration
	defaultModel string
}

// WithBaseURL sets a custom base URL
func WithBaseURL(url string) ClientOption {
	return func(cfg *clientConfig) {
		cfg.baseURL = url
	}
}

// WithTimeout sets custom HTTP timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(cfg *clientConfig) {
		cfg.timeout = timeout
	}
}

// WithDefaultModel sets the default model
func WithDefaultModel(model string) ClientOption {
	return func(cfg *clientConfig) {
		cfg.defaultModel = model
	}
}

// NewClient creates a new OpenAI-compatible client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	cfg := &clientConfig{
		baseURL:      DefaultBaseURL,
		timeout:      DefaultTimeout,
		defaultModel: ModelClaude35Sonnet,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Build client options for text client
	clientOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(cfg.baseURL),
		option.WithHTTPClient(&http.Client{Timeout: cfg.timeout}),
		option.WithHeader("HTTP-Referer", "https://github.com/rezonia/invoice-processor"),
		option.WithHeader("X-Title", "Invoice Processor"),
	}

	// Build client options for vision client with custom transport
	visionHTTPClient := &http.Client{
		Timeout: cfg.timeout,
		Transport: &visionHeaderTransport{
			base: http.DefaultTransport,
		},
	}
	visionClientOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(cfg.baseURL),
		option.WithHTTPClient(visionHTTPClient),
		option.WithHeader("HTTP-Referer", "https://github.com/rezonia/invoice-processor"),
		option.WithHeader("X-Title", "Invoice Processor"),
	}

	return &Client{
		client:       openai.NewClient(clientOpts...),
		visionClient: openai.NewClient(visionClientOpts...),
		defaultModel: cfg.defaultModel,
	}
}

// ChatText is a convenience method for text-only chat
func (c *Client) ChatText(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	if model == "" {
		model = c.defaultModel
	}

	messages := []openai.ChatCompletionMessageParamUnion{}

	if systemPrompt != "" {
		messages = append(messages, openai.SystemMessage(systemPrompt))
	}

	messages = append(messages, openai.UserMessage(userPrompt))

	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       model,
		Messages:    messages,
		MaxTokens:   param.NewOpt[int64](4096),
		Temperature: param.NewOpt[float64](0.1),
	})
	if err != nil {
		return "", fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

// ChatWithImage sends a multimodal request with an image
func (c *Client) ChatWithImage(ctx context.Context, model, systemPrompt, userPrompt string, imageData []byte, mimeType string) (string, error) {
	if model == "" {
		model = c.defaultModel
	}

	// Convert image to base64 data URL
	b64 := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	messages := []openai.ChatCompletionMessageParamUnion{}

	if systemPrompt != "" {
		messages = append(messages, openai.SystemMessage(systemPrompt))
	}

	// Multimodal message with text and image
	contentParts := []openai.ChatCompletionContentPartUnionParam{
		openai.TextContentPart(userPrompt),
		openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
			URL: dataURL,
		}),
	}
	messages = append(messages, openai.UserMessage(contentParts))

	// Use visionClient which has the Copilot-Vision-Request header
	resp, err := c.visionClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       model,
		Messages:    messages,
		MaxTokens:   param.NewOpt[int64](4096),
		Temperature: param.NewOpt[float64](0.1),
	})
	if err != nil {
		return "", fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

// ExtractJSON extracts JSON from LLM response (handles markdown code blocks)
func ExtractJSON(response string) string {
	// Try to find JSON in markdown code block
	if start := strings.Index(response, "```json"); start != -1 {
		start += 7
		if end := strings.Index(response[start:], "```"); end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}

	// Try to find JSON in generic code block
	if start := strings.Index(response, "```"); start != -1 {
		start += 3
		// Skip language identifier if present
		if nl := strings.Index(response[start:], "\n"); nl != -1 {
			start += nl + 1
		}
		if end := strings.Index(response[start:], "```"); end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}

	// Try to find raw JSON object/array
	response = strings.TrimSpace(response)
	if (strings.HasPrefix(response, "{") && strings.HasSuffix(response, "}")) ||
		(strings.HasPrefix(response, "[") && strings.HasSuffix(response, "]")) {
		return response
	}

	return response
}
