package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/kajvans/foundry/internal/detect"
	"github.com/spf13/cobra"
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect installed languages, package managers, and tools",
	Long: `Scan your system to detect common languages and tools.

This helps Foundry tailor defaults (like language and package manager) and
ensure prerequisites (like git and docker) are available.

No changes are made without your confirmation.`,
	Example: `  foundry detect`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonOut, _ := cmd.Flags().GetBool("json")
		assumeYes, _ := cmd.Flags().GetBool("yes")
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

		color.Cyan("Scanning your system...")

		// Call helper to perform detection
		result := detect.ScanSystem()

		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			_ = enc.Encode(result)
		} else {
			// Print results
			detect.PrintResult(result)
		}

		// Ask user for confirmation
		color.Green("Detection complete. Please review the detected tools above.")
		if nonInteractive {
			if assumeYes {
				color.Green("Configuration saved.")
				detect.SaveConfig(result)
			}
			return
		}

		var response string
		color.New(color.Bold).Print("Does this look correct? (y/n): ")
		fmt.Scanln(&response)
		if response == "y" {
			color.Green("Configuration saved.")
			detect.SaveConfig(result)
		} else {
			color.Yellow("Please adjust configuration manually or re-run detection.")
		}
	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
	detectCmd.Flags().Bool("json", false, "Output results in JSON format")
	detectCmd.Flags().Bool("yes", false, "Assume 'yes' when saving results (use with --non-interactive)")
	detectCmd.Flags().Bool("non-interactive", false, "Do not prompt; just print or save if --yes is provided")
}
