package cmd

import (
	"fmt"
	"os"

	"github.com/buggy-bits/repo-lens/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgPath      string
	cliMode      string
	cliOllamaURL string
)

var rootCmd = &cobra.Command{
	Use:   "lens",
	Short: "Repo Lens: Offline RAG CLI for codebase Q&A",
	Long: `Repo Lens indexes your local codebase and answers questions 
using a local LLM via Ollama. No cloud, no API keys.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			return err
		}

		// Apply persistent CLI overrides
		if cmd.Flags().Changed("mode") {
			cfg.Mode = cliMode
		}
		if cmd.Flags().Changed("ollama-url") {
			cfg.OllamaURL = cliOllamaURL
		}

		config.ActiveConfig = cfg
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&cliMode, "mode", "", "operation mode (local or online)")
	rootCmd.PersistentFlags().StringVar(&cliOllamaURL, "ollama-url", "", "Ollama API URL")
}
