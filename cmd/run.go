/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/gotray/got/cmd/internal/rungo"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [flags] [package] [arguments...]",
	Short: "Run a Go package with Python environment configured",
	Long: func() string {
		intro := "Run executes a Go package with the Python environment properly configured.\n\n"
		help, err := rungo.GetGoCommandHelp("run")
		if err != nil {
			return intro + "Failed to get go help: " + err.Error()
		}
		return intro + help
	}(),
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := rungo.RunCommand("go", append([]string{"run"}, args...)); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
