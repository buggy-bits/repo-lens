package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CodeChunk represents a chunk of logical piece of code
type CodeChunk struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
	ChunkID  string `json:"chunk_id"`
}

// Map lookup is faster than looking throughout the array for allowed extention is present or not
var allowedExtentions = map[string]bool{
	".go": true, ".js": true, ".ts": true, ".py": true, ".rs": true, ".java": true,
	".md": true, ".txt": true, ".json": true, ".yaml": true, ".yml": true,
	".css": true, ".html": true, ".sql": true, ".sh": true,
	".cpp": true, ".c": true, ".cs": true, ".php": true, ".swift": true, ".kt": true, ".dart": true, ".scala": true, ".lua": true,
	".rb": true, ".pl": true, ".r": true, ".jl": true, ".hs": true, ".erl": true, ".ex": true, ".exs": true,
}

var ignoreDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "__pycache__": true,
	".venv": true, "dist": true, "build": true, ".next": true, ".idea": true, ".vite": true,
	"out": true, "bin": true, "obj": true, "target": true, "logs": true, "tmp": true, "temp": true,
}

var splitRegex = regexp.MustCompile(`(?m)^(func |class |def |export |async |interface |struct |const |let |var )`)

func ChunkDirectory(rootPath string) ([]CodeChunk, error) {
	var chunks []CodeChunk
	var filesProcessed int

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored Directories
		if d.IsDir() {
			if ignoreDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !allowedExtentions[ext] {
			return nil
		}

		filesProcessed++

		fileContent, err := os.ReadFile(path)
		fileChunks := splitIntoChunks(string(fileContent), path)
		chunks = append(chunks, fileChunks...)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("Scanned %d files\nGenerated %d semantic chunks\n", filesProcessed, len(chunks))
	return chunks, nil
}

// Split file content into chunks

func splitIntoChunks(content string, filePath string) []CodeChunk {
	var chunks []CodeChunk
	content = strings.TrimSpace(content)

	// Very Temperry chunking algorithm
	// TODO: Should implement a good function to chunk into meaningful chunks

	if len(content) < 600 {
		return append(chunks, CodeChunk{
			FilePath: filePath,
			Content:  content,
			ChunkID:  fmt.Sprintf("%s::0", filePath),
		})
	}

	matches := splitRegex.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		// Fallback: split by double newlines
		return splitByNewlines(content, filePath)
	}

	// Create chunks between matches
	for i, match := range matches {
		start := match[0]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		chunkText := strings.TrimSpace(content[start:end])
		if chunkText == "" || len(chunkText) < 50 {
			continue
		}

		chunks = append(chunks, CodeChunk{
			FilePath: filePath,
			Content:  chunkText,
			ChunkID:  fmt.Sprintf("%s::%d", filePath, i),
		})
	}
	return chunks
}

func splitByNewlines(content string, filePath string) []CodeChunk {
	var chunks []CodeChunk
	parts := strings.Split(content, "\n\n")
	for i, part := range parts {
		if len(strings.TrimSpace(part)) < 100 {
			continue
		}
		chunks = append(chunks, CodeChunk{
			FilePath: filePath,
			Content:  strings.TrimSpace(part),
			ChunkID:  fmt.Sprintf("%s::%d", filePath, i),
		})
	}
	return chunks

}

func PrintSampleChunks(chunks []CodeChunk, limit int) {
	if len(chunks) == 0 {
		fmt.Println("No chunks generated.")
		return
	}
	fmt.Println("\nSample Chunks (Verification):")
	for i := 0; i < limit && i < len(chunks); i++ {
		fmt.Printf("--- Chunk %d (%s) ---\n%s\n\n", i+1, chunks[i].ChunkID, chunks[i].Content[:min(len(chunks[i].Content), 300)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
