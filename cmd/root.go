/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/kajvans/foundry/internal/config"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "foundry",
	Short: "Create new projects from templates and manage Foundry settings",
	Long: `Foundry helps you scaffold new projects from your own templates and manage preferences.

Key features:

  - Detect installed languages and tools (git, docker, etc.)
  - Save project templates and set per-language defaults
  - Create new projects from a chosen template
  - Interactive arrow-key menus for language and template selection
  - Manage author, license, language defaults, and more

Color output:
  - Use --no-color to disable colored output
  - Use --color to force colors (overrides NO_COLOR environment variable)
  - Set NO_COLOR environment variable to disable colors globally

Examples:

  # Create a new Go project using the default Go template
  foundry new my-api --language Go

  # Create a new project using a specific saved template
  foundry new my-app --template react-starter

  # Interactively choose language and template
  foundry new my-project

  # View current configuration
  foundry config Go

  # Disable colored output
  foundry template list --no-color
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Apply no-color flag early, before command execution
	// Check for explicit --no-color=false or --color to re-enable colors
	for _, arg := range os.Args {
		if arg == "--color" {
			color.NoColor = false
			break
		}
		if arg == "--no-color" || arg == "-no-color" {
			color.NoColor = true
			break
		}
		// Check for --no-color=true or --no-color=false
		if strings.HasPrefix(arg, "--no-color=") {
			if strings.HasSuffix(arg, "=false") || strings.HasSuffix(arg, "=0") {
				color.NoColor = false
			} else if strings.HasSuffix(arg, "=true") || strings.HasSuffix(arg, "=1") {
				color.NoColor = true
			}
			break
		}
	}

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Set version using Cobra's built-in Version field
	rootCmd.Version = version

	// Persistent flags
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().Bool("color", false, "Force colored output (overrides NO_COLOR env)")
	rootCmd.PersistentFlags().String("config", "", "Path to config file (overrides default)")

	// Respect NO_COLOR environment variable unless explicitly overridden
	if v, ok := os.LookupEnv("NO_COLOR"); ok && strings.TrimSpace(v) != "" {
		color.NoColor = true
	}

	// Use PersistentPreRun to apply global flags before any command runs
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// no-color handling (flag takes precedence over env)
		if cmd.Flags().Changed("no-color") {
			nc, _ := cmd.Flags().GetBool("no-color")
			color.NoColor = nc
		}

		// config path override
		if cmd.Flags().Changed("config") {
			path, _ := cmd.Flags().GetString("config")
			if path != "" {
				config.SetConfigPathOverride(path)
			}
		}
	}
}

// version is injected via -ldflags at build time, defaults to "dev"
var version = "dev"
