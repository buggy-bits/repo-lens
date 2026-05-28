package store

import (
	"encoding/json"
	"os"
)

type VectorChunk struct {
	ChunkID  string    `json:"chunk_id"`
	FilePath string    `json:"file_path"`
	Content  string    `json:"content"`
	Vector   []float64 `json:"vector"`
}

type VectorStore struct {
	Chunks []VectorChunk `json:"chunks"`
}

func SaveStore(path string, store *VectorStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadStore(path string) (*VectorStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &VectorStore{Chunks: []VectorChunk{}}, nil
		}
		return nil, err
	}
	var store VectorStore
	return &store, json.Unmarshal(data, &store)
}
