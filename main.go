package main

import (
	"github.com/Dreamacro/go-check/action"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Short: "go-check is a go module updater",
	Run:   action.Upgrade,
}

func main() {
	rootCmd.Execute()
}
