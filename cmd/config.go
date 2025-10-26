package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kajvans/foundry/internal/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config [language] [template]",
	Short: "View or update configuration settings",
	Long: `View or update Foundry configuration.

You can set specific values directly via flags:

  --user <name>              Set the author name
  --license <type>           Set the license (MIT, Apache, etc.)
  --default-language <l>     Set the default language for new projects
  --clear-default <lang>     Clear default template for a specific language
  --docker                   Enable Dockerfile generation
  --interactive              Enable interactive mode for project creation
  --view                     Show current configuration settings

To set a default template for a language, use positional arguments:
  foundry config <language> <template-name>
`,
	Example: `  foundry config --user "John" --docker
  foundry config --license Apache
  foundry config Go my-go-template
  foundry config Python flask-starter
  foundry config --clear-default Go
  foundry config --view`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config.PrintConfig()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Load current config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = &config.Config{} // fallback
	}

	// Define flags with defaults from config
	configCmd.Flags().String("user", cfg.Author, "Set the author name")
	configCmd.Flags().String("license", cfg.License, "Set the license type")
	configCmd.Flags().String("default-language", cfg.DefaultLanguage, "Set the default language")
	configCmd.Flags().Bool("docker", cfg.Docker, "Enable Dockerfile generation")
	configCmd.Flags().Bool("interactive", cfg.Interactive, "Enable interactive mode")
	configCmd.Flags().Bool("view", false, "Show current configuration settings")
	configCmd.Flags().String("clear-default", "", "Clear default template for a specific language")

	// TODO: Add a global --no-color flag (and respect NO_COLOR env) to disable colored output.
	// TODO: Provide shell completions for <language> and <template> positional args.

	// Provide smart completions for positional args: <language> and <template>
	configCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Load templates
		tpls, err := config.ListTemplates()
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		switch len(args) {
		case 0:
			// Suggest languages: unique set from templates plus some common values
			langSet := map[string]struct{}{
				"Go": {}, "Python": {}, "JavaScript": {}, "TypeScript": {}, "React": {}, "Rust": {}, "Java": {},
			}
			for _, t := range tpls {
				if t.Language != "" {
					langSet[t.Language] = struct{}{}
				}
			}
			var langs []string
			for l := range langSet {
				if toComplete == "" || strings.HasPrefix(strings.ToLower(l), strings.ToLower(toComplete)) {
					langs = append(langs, l)
				}
			}
			sort.Strings(langs)
			return langs, cobra.ShellCompDirectiveNoFileComp
		case 1:
			// Suggest template names
			var names []string
			for _, t := range tpls {
				if toComplete == "" || strings.HasPrefix(strings.ToLower(t.Name), strings.ToLower(toComplete)) {
					names = append(names, t.Name)
				}
			}
			sort.Strings(names)
			return names, cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// Run updates any flags passed
	configCmd.Run = func(cmd *cobra.Command, args []string) {
		if view, _ := cmd.Flags().GetBool("view"); view {
			config.PrintConfig()
			return
		}

		changed := false

		// Handle set-default with positional arguments
		// Usage: foundry config <language> <template>
		if len(args) == 2 {
			lang := args[0]
			tmpl := args[1]
			if err := config.SetLanguageDefault(lang, tmpl); err != nil {
				fmt.Fprintf(os.Stderr, "Error setting default for %s: %v\n", lang, err)
				os.Exit(1)
			}
			fmt.Printf("✓ Set default template for %s: %s\n", lang, tmpl)
			changed = true
		}

		// Handle clear-default flag
		if clearLang, _ := cmd.Flags().GetString("clear-default"); clearLang != "" {
			if err := config.ClearLanguageDefault(clearLang); err != nil {
				fmt.Fprintf(os.Stderr, "Error clearing default for %s: %v\n", clearLang, err)
				os.Exit(1)
			}
			fmt.Printf("✓ Cleared default template for %s\n", clearLang)
			changed = true
		}

		// Get flags and update config if they were provided
		if user, _ := cmd.Flags().GetString("user"); user != "" && cmd.Flags().Changed("user") {
			config.SetConfigValue("author", user)
			changed = true
		}
		if license, _ := cmd.Flags().GetString("license"); license != "" && cmd.Flags().Changed("license") {
			config.SetConfigValue("license", license)
			changed = true
		}
		if lang, _ := cmd.Flags().GetString("default-language"); lang != "" && cmd.Flags().Changed("default-language") {
			config.SetConfigValue("default_language", lang)
			changed = true
		}
		if cmd.Flags().Changed("docker") {
			docker, _ := cmd.Flags().GetBool("docker")
			config.SetConfigValue("docker", docker)
			changed = true
		}
		if cmd.Flags().Changed("interactive") {
			interactive, _ := cmd.Flags().GetBool("interactive")
			config.SetConfigValue("interactive", interactive)
			changed = true
		}

		if !changed {
			// No updates provided; show current configuration
			config.PrintConfig()
			return
		}

		fmt.Println("\nConfiguration updated. Current values:")
		config.PrintConfig()
	}

	// Use default Cobra help which includes usage, flags, and examples
}
