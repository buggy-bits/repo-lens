package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/buggy-bits/repo-lens/internal/config"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete vector_store.json and reset the index",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(config.ActiveConfig.VectorStorePath); os.IsNotExist(err) {
			fmt.Println("No vector store found. Nothing to clear.")
			return
		}

		fmt.Print("This will delete vector_store.json and remove all indexed data. Continue? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input != "y" && input != "yes" {
			fmt.Println("Cancelled. Index preserved.")
			return
		}

		if err := os.Remove(config.ActiveConfig.VectorStorePath); err != nil {
			fmt.Printf("Failed to delete store: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Vector store cleared successfully.")
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
