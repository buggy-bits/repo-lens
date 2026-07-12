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
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

const topMatches = 3
const chatModel = "qwen2.5:7b"
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

		// Header style
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

		// Print sources style
		sourceHeaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD")).
			Bold(true).
			MarginBottom(1)

		// Metrics style
		metricsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Italic(true)

		if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			fmt.Println("🔍 Searching vectors & fetching context...")
			db, err := store.LoadStore(vectorStorePath)
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				os.Exit(1)
			}
			if len(db.Chunks) == 0 {
				fmt.Println("❌ Error: index is empty")
				os.Exit(1)
			}

			queryVec, err := ollama.GetEmbedding(query)
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				os.Exit(1)
			}

			matches := vector.FindTopMatches(queryVec, db.Chunks, topMatches)
			if len(matches) == 0 {
				fmt.Println("❌ Error: no relevant chunks found")
				os.Exit(1)
			}

			var contextParts []string
			for _, m := range matches {
				contextParts = append(contextParts, fmt.Sprintf("[File %s]\n```\n%s\n```", m.FilePath, m.Content))
			}
			context := strings.Join(contextParts, "\n\n---\n\n")

			fmt.Println("🧠 Thinking and generating response...")
			started := time.Now()
			var fullResponse strings.Builder
			err = ollama.StreamChat(query, context, chatModel, func(token string) {
				fullResponse.WriteString(token)
			})
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				os.Exit(1)
			}

			duration := time.Since(started).Round(time.Second)
			responseStr := fullResponse.String()

			rendered, err := glamour.Render(responseStr, "dark")
			if err != nil {
				rendered = responseStr
			}

			fmt.Println(headerStyle.Render("💡 Repo Lens Answer:"))
			fmt.Println(rendered)

			fmt.Println(sourceHeaderStyle.Render("📚 Source Chunks:"))
			for i, match := range matches {
				fmt.Printf("  %d. %s (similarity score: %.2f)\n", i+1, match.FilePath, match.Score)
			}

			metricsStr := fmt.Sprintf("\n📊 Metrics: %.1fs | %d tokens (est.) | %d chunks matched",
				duration.Seconds(),
				int(float64(len(strings.Fields(responseStr)))*1.3),
				len(matches))
			fmt.Println(metricsStyle.Render(metricsStr))
			fmt.Println(metricsStyle.Render("────────────────────────────────────────────────────────────"))
			return
		}

		s := spinner.New()
		s.Spinner = spinner.Points
		s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))

		p := tea.NewProgram(initialAskModel(s, query))
		m, err := p.Run()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		model := m.(askModel)
		if model.err != nil {
			fmt.Printf("\n❌ Error: %v\n", model.err)
			os.Exit(1)
		}

		duration := time.Since(model.started).Round(time.Second)

		rendered, err := glamour.Render(model.response, "dark")
		if err != nil {
			rendered = model.response
		}

		fmt.Println(headerStyle.Render("💡 Repo Lens Answer:"))
		fmt.Println(rendered)

		fmt.Println(sourceHeaderStyle.Render("📚 Source Chunks:"))
		for i, match := range model.matches {
			fmt.Printf("  %d. %s (similarity score: %.2f)\n", i+1, match.FilePath, match.Score)
		}

		metricsStr := fmt.Sprintf("\n📊 Metrics: %.1fs | %d tokens (est.) | %d chunks matched",
			duration.Seconds(),
			int(float64(len(strings.Fields(model.response)))*1.3),
			len(model.matches))
		fmt.Println(metricsStyle.Render(metricsStr))
		fmt.Println(metricsStyle.Render("────────────────────────────────────────────────────────────"))
	},
}

type askModel struct {
	spinner   spinner.Model
	query     string
	retrieved bool
	context   string
	matches   []vector.RankedChunk
	response  string
	done      bool
	err       error
	started   time.Time
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
		runOnlyRetrieval(m.query),
	)
}

type retrievalResult struct {
	context string
	matches []vector.RankedChunk
	err     error
}

type askResult struct {
	response string
	err      error
}

func runOnlyRetrieval(query string) tea.Cmd {
	return func() tea.Msg {
		db, err := store.LoadStore(vectorStorePath)
		if err != nil {
			return retrievalResult{err: err}
		}
		if len(db.Chunks) == 0 {
			return retrievalResult{err: fmt.Errorf("index is empty")}
		}

		queryVec, err := ollama.GetEmbedding(query)
		if err != nil {
			return retrievalResult{err: fmt.Errorf("failed to embed query: %w", err)}
		}

		matches := vector.FindTopMatches(queryVec, db.Chunks, topMatches)
		if len(matches) == 0 {
			return retrievalResult{err: fmt.Errorf("no relevant chunks found")}
		}

		var contextParts []string
		for _, m := range matches {
			contextParts = append(contextParts, fmt.Sprintf("[File %s]\n```\n%s\n```", m.FilePath, m.Content))
		}
		context := strings.Join(contextParts, "\n\n---\n\n")

		return retrievalResult{
			context: context,
			matches: matches,
		}
	}
}

func runLLMGeneration(query, context string) tea.Cmd {
	return func() tea.Msg {
		var fullResponse strings.Builder
		err := ollama.StreamChat(query, context, chatModel, func(token string) {
			fullResponse.WriteString(token)
		})
		if err != nil {
			return askResult{err: fmt.Errorf("streaming failed: %w", err)}
		}
		return askResult{response: fullResponse.String()}
	}
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
	case retrievalResult:
		if msg.err != nil {
			m.err = msg.err
			m.done = true
			return m, tea.Quit
		}
		m.retrieved = true
		m.context = msg.context
		m.matches = msg.matches
		return m, runLLMGeneration(m.query, m.context)
	case askResult:
		m.done = true
		m.response = msg.response
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m askModel) View() string {
	if m.err != nil {
		return ""
	}
	if !m.retrieved {
		return fmt.Sprintf("\n %s Searching vectors & fetching context...\n", m.spinner.View())
	}
	if !m.done {
		return fmt.Sprintf("\n %s Thinking and generating response...\n", m.spinner.View())
	}
	return ""
}

func init() {
	rootCmd.AddCommand(askCmd)
}
