package cmd

import (
	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "file format conversions",
	Long:  `Converting files from one format to another.`,
}

func init() {
	rootCmd.AddCommand(convertCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// convertCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// convertCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
