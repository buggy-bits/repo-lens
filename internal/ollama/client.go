package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const OllamaBaseURL = "http://localhost:11434"

type EmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

func GetEmbedding(text string) ([]float64, error) {
	reqBody := EmbedRequest{
		Model:  "nomic-embed-text:v1.5",
		Prompt: text,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(OllamaBaseURL+"/api/embeddings", "application/json", bytes.NewBuffer(jsonBytes))

	if err != nil {
		return nil, fmt.Errorf("failed to reach Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API error: %s", resp.Status)
	}

	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	return embedResp.Embedding, nil
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatStreamResponse struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

func StreamChat(query string, context string, model string, callback func(string)) error {
	prompt := fmt.Sprintf(`You are an expert codebase assistant. 
Use the following context to answer the question accurately. 
If the context doesn't contain the answer, say "I couldn't find relevant information in the codebase."

## Context:
%s

## Question:
%s

Answer concisely with code examples where helpful:`, context, query)

	reqBody := ChatRequest{
		Model:    model,
		Messages: []Message{{Role: "user", Content: prompt}},
		Stream:   true,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post(OllamaBaseURL+"/api/chat", "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to reach Ollama: %w", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		var streamResp ChatStreamResponse
		if err := json.Unmarshal(line, &streamResp); err != nil {
			continue // Skip malformed JSON lines
		}

		if streamResp.Message.Content != "" {
			callback(streamResp.Message.Content)
		}
		if streamResp.Done {
			break
		}
	}
	return nil
}
