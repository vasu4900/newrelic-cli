package agent

import (
	"github.com/spf13/cobra"
)

// Command represents the apm command
var Command = &cobra.Command{
	Use:   "agent",
	Short: "Utilities for New Relic Agents",
	Long:  `Utilities for New Relic Agents`,
}
