package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/buggy-bits/repo-lens/internal/config"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Update a configuration option in config.yaml",
	Long: `Allows setting important configuration parameters like:
- model
- embedding-model (or embedding_model)
- top-results (or top_results)
- ollama-url (or ollama_url)
- vector-store-path (or vector_store_path)
- mode`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := strings.ToLower(strings.ReplaceAll(args[0], "-", "_"))
		value := args[1]

		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			fmt.Printf("❌ Failed to load config: %v\n", err)
			os.Exit(1)
		}

		switch key {
		case "model":
			cfg.Model = value
		case "embedding_model":
			cfg.EmbeddingModel = value
		case "top_results":
			val, err := strconv.Atoi(value)
			if err != nil {
				fmt.Printf("❌ Invalid value for top_results: must be an integer\n")
				os.Exit(1)
			}
			cfg.TopResults = val
		case "ollama_url":
			cfg.OllamaURL = value
		case "vector_store_path":
			cfg.VectorStorePath = value
		case "mode":
			if value != "local" && value != "online" {
				fmt.Printf("❌ Invalid value for mode: must be 'local' or 'online'\n")
				os.Exit(1)
			}
			cfg.Mode = value
		default:
			fmt.Printf("❌ Unsupported config key: %s\n", args[0])
			fmt.Println("Supported keys: model, embedding-model, top-results, ollama-url, vector-store-path, mode")
			os.Exit(1)
		}

		if err := config.SaveConfig(cfg, cfgPath); err != nil {
			fmt.Printf("❌ Failed to save config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Config updated: %s set to %s\n", args[0], value)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}
