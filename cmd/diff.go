package cmd

import (
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "compare files, entry by entry",
	Long:  `Find all the differences between two input files, and show them.`,
}

func init() {
	rootCmd.AddCommand(diffCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// convertCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// convertCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
