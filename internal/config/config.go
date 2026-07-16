package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Mode                 string `yaml:"mode"`
	Model                string `yaml:"model"`
	EmbeddingModel       string `yaml:"embedding_model"`
	TopResults           int    `yaml:"top_results"`
	VectorStorePath      string `yaml:"vector_store_path"`
	OllamaURL            string `yaml:"ollama_url"`
	OnlineProvider       string `yaml:"online_provider"`
	OnlineAPIKey         string `yaml:"online_api_key"`
	OnlineAPIBase        string `yaml:"online_api_base"`
	OnlineModel          string `yaml:"online_model"`
	OnlineEmbeddingModel string `yaml:"online_embedding_model"`
}

var ActiveConfig = &Config{
	Mode:                 "local",
	Model:                "qwen2.5:7b",
	EmbeddingModel:       "nomic-embed-text:v1.5",
	TopResults:           3,
	VectorStorePath:      "vector_store.json",
	OllamaURL:            "http://localhost:11434",
	OnlineProvider:       "openai",
	OnlineAPIKey:         "",
	OnlineAPIBase:        "",
	OnlineModel:          "gpt-4o",
	OnlineEmbeddingModel: "text-embedding-3-small",
}

const DefaultConfigTemplate = `# Repo Lens Configuration File
# 
# You can customize these values to change how Repo Lens behaves.

# Mode of operation:
#   "local"  - uses Ollama (default)
#   "online" - uses third-party APIs (for future use)
mode: "local"

# The LLM model name to use for generating answers (e.g. qwen2.5:7b, llama3)
model: "qwen2.5:7b"

# The embedding model name to use for indexing the codebase (e.g. nomic-embed-text:v1.5)
embedding_model: "nomic-embed-text:v1.5"

# Number of top matching codebase chunks to retrieve as context (default: 3)
top_results: 3

# Path to the vector database file (default: vector_store.json)
vector_store_path: "vector_store.json"

# Local Ollama connection URL
ollama_url: "http://localhost:11434"

# Online API configuration (for future use when online mode is active)
# API provider to use: "openai", "gemini", "anthropic"
online_provider: "openai"

# API key for the selected online provider
online_api_key: ""

# Optional custom endpoint URL for the online provider
online_api_base: ""

# Model name to use in online mode
online_model: "gpt-4o"

# Embedding model name to use in online mode
online_embedding_model: "text-embedding-3-small"
`

// DefaultConfigPath returns the default configuration path under UserConfigDir.
func DefaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "repo-lens", "config.yaml"), nil
}

// LoadConfig reads the configuration file at the specified path.
// If customPath is empty, it uses DefaultConfigPath().
// If the configuration file does not exist, it creates it with the default values.
func LoadConfig(customPath string) (*Config, error) {
	path := customPath
	var err error
	if path == "" {
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get user config directory: %w", err)
		}
	}

	// Check if file exists. If not, create directory and write defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory %s: %w", dir, err)
		}

		if err := os.WriteFile(path, []byte(DefaultConfigTemplate), 0644); err != nil {
			return nil, fmt.Errorf("failed to write default config file: %w", err)
		}
	}

	// Read and parse configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	cfg := *ActiveConfig // copy defaults
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	return &cfg, nil
}
