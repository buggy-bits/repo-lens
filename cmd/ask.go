package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/buggy-bits/repo-lens/internal/ollama"
	"github.com/buggy-bits/repo-lens/internal/store"
	"github.com/buggy-bits/repo-lens/internal/vector"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const topMatches = 3

// const chatModel = "qwen3:4b"   // Change to qwen:7b or phi3 if preferred
const chatModel = "qwen2.5:7b" // Change to qwen:7b or phi3 if preferred

const vectorStorePath = "vector_store.json"

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Query your indexed codebase using local AI",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		if _, err := os.Stat(vectorStorePath); os.IsNotExist(err) {
			fmt.Println("❌ No index found. Run 'lens ingest ./path' first.")
			os.Exit(1)
		}

		s := spinner.New()
		s.Spinner = spinner.Points
		s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))

		p := tea.NewProgram(initialAskModel(s, query))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

type askModel struct {
	spinner  spinner.Model
	query    string
	response string
	done     bool
	streamed bool
	err      error
	started  time.Time
}

func initialAskModel(s spinner.Model, query string) askModel {
	return askModel{
		spinner: s,
		query:   query,
		started: time.Now(),
	}
}

func (m askModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		// runRetrieval(m.query),
		runFullRAG(m.query),
	)
}

func (m askModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case askResult:
		m.done = true
		// m.context = msg.context
		m.response = msg.response
		m.err = msg.err
		// if m.err == nil && !m.streamed {
		// 	m.streamed = true
		// return m, streamResponse(m.context, m.query)
		// 	return m, tea.Quit
		// }
		return m, tea.Quit
	}
	return m, nil
}

func (m askModel) View() string {
	if !m.done {
		return fmt.Sprintf("\n %s Embedding query & searching vectors...\n", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("\nQuery failed: %v\n", m.err)
	}

	duration := time.Since(m.started).Round(time.Second)

	rendered, _ := glamour.Render(m.response, "dark")

	output := fmt.Sprintf("\n 💡 Repo Lens Answer:\n%s\n", rendered)
	output += fmt.Sprintf("\n 📊 Metrics: %.1fs | %d tokens (est.) | %d chunks matched\n",
		duration.Seconds(),
		int(float64(len(strings.Fields(m.response)))*1.3), // rough token estimation
		topMatches)
	output += "────────────────────────────────────────────────────────────\n"
	return output
}

type askResult struct {
	response string
	err      error
}

func runFullRAG(query string) tea.Cmd {
	return func() tea.Msg {
		db, err := store.LoadStore(vectorStorePath)
		if err != nil {
			return askResult{err: err}
		}
		if len(db.Chunks) == 0 {
			return askResult{err: fmt.Errorf("index is empty")}
		}

		queryVec, err := ollama.GetEmbedding(query)
		if err != nil {
			return askResult{err: fmt.Errorf("failed to embed query: %w", err)}
		}

		matches := vector.FindTopMatches(queryVec, db.Chunks, topMatches)
		if len(matches) == 0 {
			return askResult{err: fmt.Errorf("no relevant chunks found")}
		}

		var contextParts []string
		for _, m := range matches {
			contextParts = append(contextParts, fmt.Sprintf("[File %s]\n```\n%s\n```", m.FilePath, m.Content))
		}
		context := strings.Join(contextParts, "\n\n---\n\n")

		// 5. Stream & buffer response
		var fullResponse strings.Builder
		err = ollama.StreamChat(query, context, chatModel, func(token string) {
			fullResponse.WriteString(token)
		})

		if err != nil {
			return askResult{err: fmt.Errorf("streaming failed: %w", err)}
		}

		return askResult{response: fullResponse.String()}
	}
}

// func streamResponse(context, query string) tea.Cmd {
// 	return func() tea.Msg {
// 		fmt.Println("\n💡 Answer:")
// 		fmt.Println(strings.Repeat("─", 60))
// 		err := ollama.StreamChat(query, context, chatModel, func(token string) {
// 			fmt.Print(token)
// 		})
// 		fmt.Println("\n" + strings.Repeat("─", 60))
// 		if err != nil {
// 			return askResult{err: fmt.Errorf("streaming error: %w", err)}
// 		}
// 		return askResult{context: "", err: nil} // Signal completion
// 	}
// }

func init() {
	rootCmd.AddCommand(askCmd)
}
