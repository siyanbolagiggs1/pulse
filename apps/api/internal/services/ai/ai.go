// Package ai wraps the LLM providers used by the support-chat assistant.
// Groq is the primary chat-completion provider (fast, generous free tier);
// Gemini is the fallback for chat completions and the only embeddings
// provider (Groq has no embeddings endpoint). Both are called over plain
// HTTP — neither has a canonical Go SDK, and the request/response shapes are
// small enough that a dependency isn't worth it.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/pulse/api/internal/config"
)

var httpClient = &http.Client{Timeout: 20 * time.Second}

var ErrNoProvider = errors.New("no AI provider configured")

// Reply generates a chat completion for a single system+user turn. Groq is
// tried first; Gemini is used if Groq is unconfigured or the request fails.
func Reply(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if config.App.GroqAPIKey != "" {
		reply, err := replyGroq(ctx, systemPrompt, userMessage)
		if err == nil {
			return reply, nil
		}
	}
	if config.App.GeminiAPIKey != "" {
		return replyGemini(ctx, systemPrompt, userMessage)
	}
	return "", ErrNoProvider
}

func replyGroq(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	payload, _ := json.Marshal(map[string]any{
		"model": config.App.GroqModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature": 0.3,
		"max_tokens":  300,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+config.App.GroqAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("groq error %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("groq: empty response")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func replyGemini(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		config.App.GeminiModel, config.App.GeminiAPIKey)

	payload, _ := json.Marshal(map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]string{{"text": userMessage}}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.3,
			"maxOutputTokens": 300,
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("gemini error %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini: empty response")
	}
	return strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text), nil
}

// Embed returns a semantic embedding vector for text. Gemini is the only
// embeddings provider wired up — Groq doesn't expose an embeddings endpoint.
func Embed(ctx context.Context, text string) ([]float32, error) {
	if config.App.GeminiAPIKey == "" {
		return nil, ErrNoProvider
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s",
		config.App.GeminiEmbeddingModel, config.App.GeminiAPIKey)

	payload, _ := json.Marshal(map[string]any{
		"content": map[string]any{
			"parts": []map[string]string{{"text": text}},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gemini embed error %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	return parsed.Embedding.Values, nil
}

// CosineSimilarity returns the cosine similarity of two equal-length vectors
// in [-1, 1]. Returns 0 for empty or mismatched-length inputs.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
