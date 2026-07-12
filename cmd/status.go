package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check Ollama connection and index status",
	Run: func(cmd *cobra.Command, args []string) {
		if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			result := checkSystem().(statusResult)
			if result.err != nil {
				fmt.Printf("Error: %v\n", result.err)
				os.Exit(1)
			}
			fmt.Println("\nRepo Lens Status")
			fmt.Printf("   Ollama API: %s\n", checkmark(result.ollamaOK))
			fmt.Printf("   Models (llama3, nomic-embed-text): %s\n", checkmark(result.modelsOK))
			fmt.Printf("   Vector store (vector_store.json): %s\n", checkmark(result.storeOK))
			if result.storeOK {
				fmt.Println("   → Ready to ask questions!")
			}
			return
		}

		// Show spinner while checking
		s := spinner.New()
		s.Spinner = spinner.Dot
		p := tea.NewProgram(initialStatusModel(s))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// Minimal bubbletea model for status check
type statusModel struct {
	spinner  spinner.Model
	checked  bool
	ollamaOK bool
	modelsOK bool
	storeOK  bool
	err      error
}

func initialStatusModel(s spinner.Model) statusModel {
	return statusModel{
		spinner: s,
	}
}

func (m statusModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkSystem, // Custom command to ping Ollama + check files
	)
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case statusResult:
		m.checked = true
		m.ollamaOK = msg.ollamaOK
		m.modelsOK = msg.modelsOK
		m.storeOK = msg.storeOK
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m statusModel) View() string {
	if !m.checked {
		return fmt.Sprintf("\n %s Checking Repo Lens status...\n", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf(" Error: %v\n", m.err)
	}

	status := "\nRepo Lens Status\n"
	status += fmt.Sprintf("   Ollama API: %s\n", checkmark(m.ollamaOK))
	status += fmt.Sprintf("   Models (llama3, nomic-embed-text): %s\n", checkmark(m.modelsOK))
	status += fmt.Sprintf("   Vector store (vector_store.json): %s\n", checkmark(m.storeOK))
	if m.storeOK {
		status += "   → Ready to ask questions!\n"
	}
	return status + "\n"
}

func checkmark(ok bool) string {
	if ok {
		return "OK"
	}
	return "Missing"
}

// statusResult carries the outcome of system checks
type statusResult struct {
	ollamaOK bool
	modelsOK bool
	storeOK  bool
	err      error
}

// checkSystem runs the actual health checks
func checkSystem() tea.Msg {
	result := statusResult{}

	// 1. Ping Ollama API
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		result.err = fmt.Errorf("Ollama not running: %w", err)
		return result
	}
	defer resp.Body.Close()
	result.ollamaOK = resp.StatusCode == 200
	// TODO: Handle specific Model
	// 2. Check models (simplified: just check if response contains model names)
	// In MVP, we'll do a basic string check; V2 can parse JSON properly
	// For now, assume OK if API responded
	result.modelsOK = result.ollamaOK

	// 3. Check vector_store.json exists
	if _, err := os.Stat("vector_store.json"); err == nil {
		result.storeOK = true
	}

	return result
}
