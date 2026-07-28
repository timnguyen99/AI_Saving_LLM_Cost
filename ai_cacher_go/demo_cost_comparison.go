package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/ai_cacher_go"
)

type MockOpenAIClient struct{}

type DemoResult struct {
	Payload    map[string]any
	CachedBy   string
	LatencyMs  int64
	SelectedModel string
}

func (c *MockOpenAIClient) CreateChatCompletion(messages []ai_cacher_go.ChatMessage, params map[string]any) (map[string]any, error) {
	prompt := ""
	for _, msg := range messages {
		if msg.Role == "user" {
			prompt += msg.Content
		}
	}
	responseText := fmt.Sprintf("AI response for prompt: %s", prompt)
	promptTokens := len(strings.Fields(prompt))
	if promptTokens == 0 {
		promptTokens = 1
	}
	completionTokens := len(strings.Fields(responseText))
	if completionTokens == 0 {
		completionTokens = 1
	}
	return map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"role":    "assistant",
					"content": responseText,
				},
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
		},
	}, nil
}

func countTokens(payload map[string]any) int {
	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		return 0
	}
	promptTokens, _ := usage["prompt_tokens"].(int)
	completionTokens, _ := usage["completion_tokens"].(int)
	return promptTokens + completionTokens
}

func runDirect(client *MockOpenAIClient, messages []ai_cacher_go.ChatMessage) (map[string]any, error) {
	result, err := client.CreateChatCompletion(messages, nil)
	if err != nil {
		return nil, err
	}
	result["cached_by"] = "Direct-Cloud"
	result["latency_ms"] = int64(1500)
	return result, nil
}

func runCached(wrapper *ai_cacher_go.AICacherWrapper, messages []ai_cacher_go.ChatMessage) (map[string]any, error) {
	return wrapper.CreateCompletion(context.Background(), messages, map[string]any{})
}

func main() {
	messagesBatch := [][]ai_cacher_go.ChatMessage{
		{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Summarize the monthly sales report."},
		},
		{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Summarize the monthly sales report."},
		},
		{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Convert this list to JSON: apple, banana, cherry."},
		},
		{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Summarize the monthly sales report."},
		},
		{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Convert this list to JSON: apple, banana, cherry."},
		},
	}

	client := &MockOpenAIClient{}
	router := ai_cacher_go.NewSemanticCacheRouter(0.88, nil)
	wrapper := ai_cacher_go.NewAICacherWrapper(client, router)

	directTokens, cachedCloudTokens, cachedHits, directCalls, cachedCalls := 0, 0, 0, 0, 0

	fmt.Println("=== Direct cloud usage ===")
	for _, messages := range messagesBatch {
		result, err := runDirect(client, messages)
		if err != nil {
			panic(err)
		}
		directCalls++
		tokens := countTokens(result)
		directTokens += tokens
		fmt.Printf("Call %d: prompt='%s' tokens=%d\n", directCalls, messages[len(messages)-1].Content, tokens)
	}

	fmt.Println("\n=== Cached usage ===")
	for _, messages := range messagesBatch {
		result, err := runCached(wrapper, messages)
		if err != nil {
			panic(err)
		}
		if result["cached_by"] == "AI-Cacher-Local" {
			cachedHits++
		} else {
			cachedCalls++
			cachedCloudTokens += countTokens(result)
		}
		fmt.Printf(
			"Request: '%s' cached_by=%v tokens=%d\n",
			messages[len(messages)-1].Content,
			result["cached_by"],
			func() int {
				if result["cached_by"] == "AI-Cacher-Local" {
					return 0
				}
				return countTokens(result)
			}(),
		)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Printf("Direct cloud calls: %d\n", directCalls)
	fmt.Printf("Cached cloud calls: %d\n", cachedCalls)
	fmt.Printf("Local cache hits: %d\n", cachedHits)
	fmt.Printf("Total direct tokens: %d\n", directTokens)
	fmt.Printf("Total cached cloud tokens: %d\n", cachedCloudTokens)
	fmt.Printf("Token reduction: %d tokens\n", directTokens-cachedCloudTokens)
	fmt.Printf(
		"Estimated cost savings: $%.6f (assuming $0.002 per 1,000 tokens)\n",
		float64(directTokens-cachedCloudTokens)*0.000002,
	)
}
