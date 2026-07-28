// Idea and Co-author: Tim Nguyen (timothynnguyen9@gmail.com)

package ai_cacher_go

import (
	"context"
	"errors"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIClient interface {
	CreateChatCompletion(messages []ChatMessage, params map[string]any) (map[string]any, error)
}

type AICacherWrapper struct {
	Client          OpenAIClient
	Router          *SemanticCacheRouter
	SimpleModelName string
	ComplexModelName string
	Classifier      *ComplexityClassifier
}

func NewAICacherWrapper(client OpenAIClient, router *SemanticCacheRouter) *AICacherWrapper {
	return &AICacherWrapper{
		Client:          client,
		Router:          router,
		SimpleModelName: "gpt-mini",
		ComplexModelName: "gpt-4",
		Classifier:      NewComplexityClassifier(),
	}
}

func (w *AICacherWrapper) chooseModel(messages []ChatMessage, params map[string]any) string {
	if _, ok := params["model"]; ok {
		return ""
	}
	if _, ok := params["model_name"]; ok {
		return ""
	}

	userPrompt := ""
	for _, msg := range messages {
		if msg.Role == "user" {
			userPrompt += msg.Content
		}
	}

	if w.Classifier.IsLocalTask(userPrompt, "") {
		return w.SimpleModelName
	}
	return w.ComplexModelName
}

func (w *AICacherWrapper) CreateCompletion(ctx context.Context, messages []ChatMessage, params map[string]any) (map[string]any, error) {
	userPrompt := ""
	for _, msg := range messages {
		if msg.Role == "user" {
			userPrompt += msg.Content
		}
	}

	cached, err := w.Router.CheckCache(ctx, userPrompt)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		cachedCopy := make(map[string]any)
		for k, v := range cached {
			cachedCopy[k] = v
		}
		cachedCopy["cached_by"] = "AI-Cacher-Local"
		cachedCopy["latency_ms"] = 5.0
		return cachedCopy, nil
	}

	if w.Client == nil {
		return nil, errors.New("OpenAI client is required")
	}

	selectedModel := w.chooseModel(messages, params)
	if selectedModel != "" {
		params["model"] = selectedModel
	}

	start := time.Now()
	cloudResponse, err := w.Client.CreateChatCompletion(messages, params)
	if err != nil {
		return nil, err
	}
	duration := time.Since(start).Milliseconds()
	cloudResponse["cached_by"] = "Cloud-Server"
	cloudResponse["latency_ms"] = duration

	if err := w.Router.UpdateCache(ctx, userPrompt, cloudResponse); err != nil {
		return nil, err
	}

	return cloudResponse, nil
}
