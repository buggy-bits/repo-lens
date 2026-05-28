package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lens",
	Short: "Repo Lens: Offline RAG CLI for codebase Q&A",
	Long: `Repo Lens indexes your local codebase and answers questions 
using a local LLM via Ollama. No cloud, no API keys.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
