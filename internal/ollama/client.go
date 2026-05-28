package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
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
