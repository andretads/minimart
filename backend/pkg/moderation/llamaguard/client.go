package llamaguard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"minimart/pkg/moderation"
	"net/http"
	"strings"
	"time"
)

const (
	maxContentLength = 10000 // characters
	defaultTimeout   = 30 * time.Second
)

type Config struct {
	Endpoint string        // e.g., "http://localhost:11434"
	Model    string        // e.g., "llama-guard3:1b"
	Timeout  time.Duration // e.g., 30s
}

type Client struct {
	httpClient *http.Client
	endpoint   string
	model      string
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if cfg.Model == "" {
		return nil, errors.New("model is required")
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	client := &Client{
		httpClient: &http.Client{Timeout: timeout},
		endpoint:   cfg.Endpoint,
		model:      cfg.Model,
	}

	slog.Info("llamaguard client configured",
		"endpoint", client.endpoint,
		"model", client.model,
		"timeout_ms", timeout.Milliseconds(),
	)

	return client, nil
}

func (c *Client) Name() string {
	return c.model
}

func (c *Client) Moderate(ctx context.Context, content string) (moderation.Result, error) {
	start := time.Now()

	slog.DebugContext(ctx, "moderation request started",
		"content_length", len(content),
		"model", c.model,
		"endpoint", c.endpoint,
	)

	if err := validateContent(content); err != nil {
		slog.WarnContext(ctx, "moderation content too long",
			"content_length", len(content),
			"max_length", maxContentLength,
		)
		return moderation.Result{}, err
	}

	resp, err := c.send(ctx, c.request(content))
	if err != nil {
		slog.ErrorContext(ctx, "moderation request failed",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return moderation.Result{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	slog.DebugContext(ctx, "moderation response received",
		"status", resp.StatusCode,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	ollamaResp, err := decodeOllamaResponse(resp)
	if err != nil {
		slog.ErrorContext(ctx, "moderation response unusable",
			"error", err,
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return moderation.Result{}, err
	}

	result, err := parseResponse(ollamaResp.Response, c.model)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse moderation response",
			"error", err,
			"raw_response_preview", truncateString(ollamaResp.Response, 200),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return moderation.Result{}, err
	}

	slog.InfoContext(ctx, "moderation completed",
		"decision", result.Decision,
		"confidence", result.Confidence,
		"categories", result.Categories,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}

// validateContent rejects content the provider cannot moderate.
func validateContent(content string) error {
	if len(content) > maxContentLength {
		return fmt.Errorf("%w: %d > %d", moderation.ErrContentTooLong, len(content), maxContentLength)
	}
	return nil
}

// request builds the Ollama generation request for the content.
func (c *Client) request(content string) OllamaRequest {
	return OllamaRequest{
		Model:  c.model,
		Prompt: buildPrompt(content),
		Stream: false,
		Options: map[string]any{
			"temperature": 0.0,
			"top_p":       0.9,
		},
	}
}

// send posts the request and returns the raw provider response.
func (c *Client) send(ctx context.Context, reqBody OllamaRequest) (*http.Response, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.endpoint + "/api/generate"
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if reqErr != nil {
		return nil, fmt.Errorf("create request: %w", reqErr)
	}
	req.Header.Set("Content-Type", "application/json")

	slog.DebugContext(ctx, "sending moderation request",
		"url", url,
		"method", http.MethodPost,
		"request_size", len(body),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", moderation.ErrProviderUnavailable, err)
	}
	return resp, nil
}

// decodeOllamaResponse reads the response body, rejecting non-OK statuses.
func decodeOllamaResponse(resp *http.Response) (OllamaResponse, error) {
	if resp.StatusCode != http.StatusOK {
		return OllamaResponse{}, decodeError(resp)
	}
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return OllamaResponse{}, fmt.Errorf("%w: %w", moderation.ErrInvalidResponse, err)
	}
	return ollamaResp, nil
}

// decodeError builds the error for a non-OK response, preferring the
// provider's own error message when the body carries one.
func decodeError(resp *http.Response) error {
	var errResp OllamaError
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("moderation request failed with status %d", resp.StatusCode)
	}
	return fmt.Errorf("%w: %s", moderation.ErrProviderUnavailable, errResp.Error)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type OllamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options,omitempty"`
}

type OllamaResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

type OllamaError struct {
	Error string `json:"error"`
}

// newResult builds the base result for the configured model, splitting a
// "model:tag" value into name and version.
func newResult(model string) moderation.Result {
	name, version, _ := strings.Cut(model, ":")
	return moderation.Result{
		ModeratedAt:  time.Now().UTC(),
		Model:        name,
		ModelVersion: version,
	}
}

// parseResponse parses the model output, trying JSON first and falling back
// to Llama Guard's plain-text "safe"/"unsafe" format.
func parseResponse(response string, model string) (moderation.Result, error) {
	slog.Debug("parsing moderation response",
		"response_preview", truncateString(response, 200),
	)

	result := newResult(model)

	var parsed struct {
		Decision    string   `json:"decision"`
		Confidence  float64  `json:"confidence"`
		Categories  []string `json:"categories"`
		Explanation string   `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &parsed); err == nil {
		result.Decision = normalizeDecision(parsed.Decision)
		result.Confidence = parsed.Confidence
		result.Categories = parsed.Categories
		result.Explanation = parsed.Explanation

		slog.Debug("parsed moderation response as JSON",
			"decision", result.Decision,
			"confidence", result.Confidence,
			"categories", result.Categories,
		)
		return result, nil
	}

	slog.Debug("JSON parsing failed, falling back to text parsing",
		"response_preview", truncateString(response, 200),
	)

	return parseTextResponse(response, model)
}

// parseTextResponse parses Llama Guard's "safe" or "unsafe\n<categories>"
// plain-text output.
func parseTextResponse(response string, model string) (moderation.Result, error) {
	result := newResult(model)

	lines := strings.Split(strings.TrimSpace(response), "\n")
	firstLine := strings.ToLower(strings.TrimSpace(lines[0]))

	slog.Debug("parsing moderation text response",
		"first_line", firstLine,
		"total_lines", len(lines),
	)

	switch firstLine {
	case "safe":
		result.Decision = "approved"
		result.Confidence = 0.9
		result.Explanation = "Content passed moderation checks"
	case "unsafe":
		result.Decision = "rejected"
		result.Confidence = 0.9
		if len(lines) > 1 {
			result.Categories = parseCategories(lines[1])
			result.Explanation = "Content flagged: " + strings.Join(result.Categories, ", ")
		} else {
			result.Categories = []string{"unknown"}
			result.Explanation = "Content flagged by moderation model"
		}
	default:
		// Check for "needs_review" or ambiguous responses
		if strings.Contains(firstLine, "review") || strings.Contains(firstLine, "unsure") {
			result.Decision = "needs_review"
			result.Confidence = 0.5
			result.Explanation = "Model uncertain, requires human review"
		} else {
			slog.Warn("unexpected moderation response format",
				"first_line", firstLine,
				"response_preview", truncateString(response, 200),
			)
			return moderation.Result{}, fmt.Errorf("%w: unexpected response: %s", moderation.ErrInvalidResponse, response)
		}
	}

	slog.Debug("parsed moderation response as text",
		"decision", result.Decision,
		"confidence", result.Confidence,
		"categories", result.Categories,
	)

	return result, nil
}

func parseCategories(line string) []string {
	// Parse comma-separated categories from Llama Guard output
	parts := strings.Split(line, ",")
	cats := make([]string, 0, len(parts))
	for _, p := range parts {
		cat := strings.TrimSpace(strings.ToLower(p))
		if cat != "" {
			cats = append(cats, cat)
		}
	}
	return cats
}

func normalizeDecision(decision string) string {
	switch strings.ToLower(decision) {
	case "approved", "safe", "allow":
		return "approved"
	case "rejected", "unsafe", "deny", "block":
		return "rejected"
	case "needs_review", "review", "unsure":
		return "needs_review"
	default:
		return "needs_review"
	}
}

func buildPrompt(content string) string {
	// System prompt for Llama Guard 3 optimized for product reviews
	// Using the official Llama Guard conversation format
	return fmt.Sprintf(`<|begin_of_text|><|start_header_id|>system<|end_header_id|>
You are a content moderation system for an e-commerce platform. Analyze the following customer product review and determine if it violates our moderation policy.

MODERATION POLICY:
- REJECT content containing: profanity, hate speech, harassment, threats, violence, explicit sexual content, spam/advertising, phishing/malicious links, personal information (PII)
- APPROVE legitimate product feedback, including negative reviews about product quality, delivery, seller service, or purchasing experience
- NEEDS_REVIEW for ambiguous cases where you cannot confidently decide

RESPOND WITH A SINGLE JSON OBJECT:
{
  "decision": "approved" | "rejected" | "needs_review",
  "confidence": 0.0-1.0,
  "categories": ["category1", "category2"],
  "explanation": "Brief explanation of decision"
}

CATEGORIES: profanity, hate_speech, harassment, threats, violence, sexual_content, spam, phishing, pii
<|eot_id|><|start_header_id|>user<|end_header_id|>
Review: %s
<|eot_id|><|start_header_id|>assistant<|end_header_id|>
`, content)
}
