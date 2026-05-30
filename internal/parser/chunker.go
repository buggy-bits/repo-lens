package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CodeChunk represents a chunk of logical piece of code
type CodeChunk struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
	ChunkID  string `json:"chunk_id"`
}

const (
	targetChunkSize = 1500       // Target size in characters (approx 350-400 tokens)
	overlapSize     = 300        // Overlap size in characters
	maxFileSize     = 500 * 1024 // Skip files larger than 500KB (e.g. data, logs, lockfiles)
)

var allowedExtentions = map[string]bool{
	".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".py": true, ".rs": true, ".java": true, ".cpp": true, ".c": true,
	".cs": true, ".php": true, ".swift": true, ".kt": true, ".dart": true,
	".rb": true, ".pl": true, ".pm": true, ".r": true, ".jl": true,
	".hs": true, ".erl": true, ".ex": true, ".exs": true, ".scala": true,
	".lua": true, ".sh": true, ".bat": true, ".ps1": true,
	".md": true, ".txt": true, ".json": true, ".yaml": true, ".yml": true,
	".toml": true, ".xml": true, ".html": true, ".css": true, ".scss": true,
	".sql": true, ".graphql": true, ".gql": true, ".prisma": true,
	".vue": true, ".svelte": true, ".env": true, ".ini": true,
	".config": true,
}

var ignoreDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "__pycache__": true,
	".venv": true, "dist": true, "build": true, ".next": true, ".idea": true, ".vite": true,
	"out": true, "bin": true, "obj": true, "target": true, "logs": true, "tmp": true, "temp": true,
}

var ignoreFiles = map[string]bool{
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"bun.lockb":         true,
	"go.sum":            true,
	"cargo.lock":        true,
	"composer.lock":     true,
	"poetry.lock":       true,
	"pipfile.lock":      true,
	"gemfile.lock":      true,
}

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

		fileName := d.Name()
		if ignoreFiles[fileName] {
			return nil
		}

		// Check file size
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > maxFileSize {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		isAllowed := allowedExtentions[ext]
		// Handle common files without extensions
		if !isAllowed && (fileName == "Dockerfile" || fileName == "Makefile") {
			isAllowed = true
		}

		if !isAllowed {
			return nil
		}

		filesProcessed++

		fileContent, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we cannot read
		}

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

// splitIntoChunks splits content into chunks using a line-based sliding window.
// This is language-agnostic, efficient, and ensures chunks do not exceed a maximum size limit.
func splitIntoChunks(content string, filePath string) []CodeChunk {
	var chunks []CodeChunk
	content = strings.TrimSpace(content)
	if content == "" {
		return chunks
	}

	// If the entire file content is within the target chunk size, keep it as a single chunk.
	if len(content) <= targetChunkSize {
		return append(chunks, CodeChunk{
			FilePath: filePath,
			Content:  content,
			ChunkID:  fmt.Sprintf("%s::0", filePath),
		})
	}

	lines := strings.Split(content, "\n")
	var currentChunkLines []string
	currentChunkLen := 0
	chunkIdx := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		lineLen := len(line) + 1 // +1 for the newline character

		// If a single line itself is longer than targetChunkSize (e.g. minified line or massive string)
		if lineLen > targetChunkSize {
			// If we have accumulated lines, flush them first
			if len(currentChunkLines) > 0 {
				chunks = append(chunks, CodeChunk{
					FilePath: filePath,
					Content:  strings.Join(currentChunkLines, "\n"),
					ChunkID:  fmt.Sprintf("%s::%d", filePath, chunkIdx),
				})
				chunkIdx++
				currentChunkLines = nil
				currentChunkLen = 0
			}

			// Split this massive line into character chunks of targetChunkSize
			runes := []rune(line)
			for start := 0; start < len(runes); start += targetChunkSize - overlapSize {
				end := start + targetChunkSize
				if end > len(runes) {
					end = len(runes)
				}
				chunkText := string(runes[start:end])
				chunks = append(chunks, CodeChunk{
					FilePath: filePath,
					Content:  chunkText,
					ChunkID:  fmt.Sprintf("%s::%d", filePath, chunkIdx),
				})
				chunkIdx++
				if end == len(runes) {
					break
				}
			}
			continue
		}

		// If adding this line would exceed our target chunk size
		if currentChunkLen+lineLen > targetChunkSize && len(currentChunkLines) > 0 {
			// Save the current chunk
			chunks = append(chunks, CodeChunk{
				FilePath: filePath,
				Content:  strings.Join(currentChunkLines, "\n"),
				ChunkID:  fmt.Sprintf("%s::%d", filePath, chunkIdx),
			})
			chunkIdx++

			// Backtrack to create overlap
			var overlapLines []string
			overlapLen := 0
			for j := len(currentChunkLines) - 1; j >= 0; j-- {
				l := currentChunkLines[j]
				lLen := len(l) + 1
				// Always include at least one line for overlap, but check if we exceed overlapSize
				if len(overlapLines) > 0 && overlapLen+lLen > overlapSize {
					break
				}
				overlapLines = append([]string{l}, overlapLines...)
				overlapLen += lLen
			}

			currentChunkLines = overlapLines
			currentChunkLen = overlapLen
		}

		currentChunkLines = append(currentChunkLines, line)
		currentChunkLen += lineLen
	}

	// Flush any remaining lines
	if len(currentChunkLines) > 0 {
		chunks = append(chunks, CodeChunk{
			FilePath: filePath,
			Content:  strings.Join(currentChunkLines, "\n"),
			ChunkID:  fmt.Sprintf("%s::%d", filePath, chunkIdx),
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
