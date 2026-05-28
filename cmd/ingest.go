package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buggy-bits/repo-lens/internal/parser"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest [directory]",
	Short: "Scan & chunk a codebase for RAG indexing",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetPath := args[0]
		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid path: %v\n", err)
			os.Exit(1)
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Directory not found: %s\n", absPath)
			os.Exit(1)
		}

		s := spinner.New()
		s.Spinner = spinner.Points
		s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))

		p := tea.NewProgram(initialIngestModel(s, absPath))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

type ingestModel struct {
	spinner spinner.Model
	path    string
	status  string
	chunks  []parser.CodeChunk
	done    bool
	err     error
}

func initialIngestModel(s spinner.Model, path string) ingestModel {
	return ingestModel{
		spinner: s,
		path:    path,
		status:  "Scanning files & chunking logic...",
	}
}
func (m ingestModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		runIngestion(m.path),
	)
}

func (m ingestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case ingestResult:
		m.done = true
		m.chunks = msg.chunks
		m.status = msg.status
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m ingestModel) View() string {
	if !m.done {
		return fmt.Sprintf("\n %s %s\n", m.spinner.View(), m.status)
	}
	if m.err != nil {
		return fmt.Sprintf("\nIngestion failed: %v\n", m.err)
	}

	output := fmt.Sprintf("\n Ingestion Complete!\n   Path: %s\n   Chunks: %d\n", m.path, len(m.chunks))
	parser.PrintSampleChunks(m.chunks, 2)
	return output
}

type ingestResult struct {
	chunks []parser.CodeChunk
	status string
	err    error
}

func runIngestion(path string) tea.Cmd {
	return func() tea.Msg {
		chunks, err := parser.ChunkDirectory(path)
		if err != nil {
			return ingestResult{err: err}
		}
		return ingestResult{chunks: chunks, status: "Chunking complete."}
	}
}

func init() {
	rootCmd.AddCommand(ingestCmd)
}
