package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/kajvans/foundry/internal/config"
	"github.com/kajvans/foundry/internal/template"
	"github.com/spf13/cobra"
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage project templates",
	Long: `Manage custom project templates for use with 'foundry new'.
	
Templates are detected automatically based on their file structure and extensions.
You can add, list, and remove templates.`,
}

// templateAddCmd adds a new template
var templateAddCmd = &cobra.Command{
	Use:   "add <name> <path>",
	Short: "Add a new template from a directory",
	Long: `Scan a directory and save it as a reusable template.
	The language will be automatically detected based on file extensions.

	You can override the detected language tag with --language to label frameworks like React or Vue.

	Example:
  foundry template add my-go-api ./my-api-template
	foundry template add react-starter ~/templates/react-app --description "React with TypeScript" --language React`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		path := args[1]

		// Validate that 'path' exists and is a directory
		if info, err := os.Stat(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot access path: %v\n", err)
			os.Exit(1)
		} else if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: path is not a directory: %s\n", path)
			os.Exit(1)
		}

		description, _ := cmd.Flags().GetString("description")
		overrideLang, _ := cmd.Flags().GetString("language")

		// Validate template name
		if err := template.ValidateName(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// TODO: Support an optional ignore file (e.g., .foundryignore) when scanning to exclude files/dirs.
		// Scan and create template
		color.Cyan("Scanning template directory: %s", path)
		tmpl, err := template.ScanTemplate(name, path, description)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning template: %v\n", err)
			os.Exit(1)
		}

		// If user provided an override language/framework tag, apply it
		if strings.TrimSpace(overrideLang) != "" {
			tmpl.Language = strings.TrimSpace(overrideLang)
		}

		color.Green("✓ Detected language: %s", tmpl.Language)
		color.Green("✓ Found %d files", len(tmpl.Files))

		// Save to config
		configTmpl := config.Template{
			Name:        tmpl.Name,
			Path:        tmpl.Path,
			Language:    tmpl.Language,
			Description: tmpl.Description,
			Files:       tmpl.Files,
		}

		if err := config.AddTemplate(configTmpl); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving template: %v\n", err)
			os.Exit(1)
		}

		color.Green("\n✓ Template '%s' saved successfully!", name)
		fmt.Printf("  Path: %s\n", tmpl.Path)
		fmt.Printf("  Language: %s\n", tmpl.Language)
		if description != "" {
			fmt.Printf("  Description: %s\n", description)
		}
	},
}

// templateListCmd lists all saved templates
var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved templates",
	Long:  `Display all templates that have been saved and are available for use with 'foundry new'.`,
	Run: func(cmd *cobra.Command, args []string) {
		templates, err := config.ListTemplates()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading templates: %v\n", err)
			os.Exit(1)
		}

		if len(templates) == 0 {
			fmt.Println("No templates saved yet.")
			fmt.Println("\nAdd a template with: foundry template add <name> <path>")
			return
		}

		// Sorting and quiet options
		sortBy, _ := cmd.Flags().GetString("sort")
		quiet, _ := cmd.Flags().GetBool("quiet")

		switch sortBy {
		case "language":
			sort.Slice(templates, func(i, j int) bool {
				if templates[i].Language == templates[j].Language {
					return templates[i].Name < templates[j].Name
				}
				return templates[i].Language < templates[j].Language
			})
		default:
			// name
			sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
		}

		if quiet {
			for _, t := range templates {
				fmt.Println(t.Name)
			}
			return
		}

		color.New(color.Bold).Printf("Saved Templates (%d):\n\n", len(templates))
		for i, t := range templates {
			fmt.Printf("%d. %s\n", i+1, t.Name)
			fmt.Printf("   Language: %s\n", t.Language)
			fmt.Printf("   Path: %s\n", t.Path)
			if t.Description != "" {
				fmt.Printf("   Description: %s\n", t.Description)
			}
			fmt.Printf("   Files: %d\n", len(t.Files))

			// Check if this is a default template for any language
			defaultLangs := config.IsDefaultTemplate(t.Name)
			if len(defaultLangs) > 0 {
				color.Cyan("   ⭐ Default for: %v", defaultLangs)
			}

			// Check if path still exists
			if _, err := os.Stat(t.Path); os.IsNotExist(err) {
				color.Yellow("   ⚠  Warning: Path no longer exists")
			}
			fmt.Println()
		}
	},
}

// templateRemoveCmd removes a template
var templateRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a saved template",
	Long:  `Remove a template from the saved templates list. This does not delete the actual files.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		// Warn if template is default for any language
		force, _ := cmd.Flags().GetBool("force")
		if langs := config.IsDefaultTemplate(name); len(langs) > 0 && !force {
			fmt.Fprintf(os.Stderr, "Error: template '%s' is the default for: %v\nUse --force to remove it anyway.\n", name, langs)
			os.Exit(1)
		}

		if err := config.RemoveTemplate(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		color.Green("✓ Template '%s' removed successfully", name)
	},
}

// templateShowCmd shows details of a specific template
var templateShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a specific template",
	Long:  `Display detailed information about a saved template, including all files.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		tmpl, err := config.GetTemplate(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		filesOnly, _ := cmd.Flags().GetBool("files-only")
		summaryOnly, _ := cmd.Flags().GetBool("summary")
		jsonOut, _ := cmd.Flags().GetBool("json")

		if jsonOut {
			// Print full template as JSON
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			_ = enc.Encode(tmpl)
			return
		}

		if !filesOnly {
			fmt.Printf("Template: %s\n", tmpl.Name)
			fmt.Printf("Language: %s\n", tmpl.Language)
			fmt.Printf("Path: %s\n", tmpl.Path)
			if tmpl.Description != "" {
				fmt.Printf("Description: %s\n", tmpl.Description)
			}
		}

		// Check if this is a default template for any language
		defaultLangs := config.IsDefaultTemplate(name)
		if len(defaultLangs) > 0 {
			color.Cyan("Default for: %v\n", defaultLangs)
		}

		// Check if path exists
		if !filesOnly {
			if _, err := os.Stat(tmpl.Path); os.IsNotExist(err) {
				color.Yellow("\n⚠  Warning: Template path no longer exists")
			}
		}

		if summaryOnly {
			return
		}

		fmt.Printf("\nFiles (%d):\n", len(tmpl.Files))

		// Group files by directory
		dirMap := make(map[string][]string)
		for _, f := range tmpl.Files {
			dir := filepath.Dir(f)
			if dir == "." {
				dir = "(root)"
			}
			dirMap[dir] = append(dirMap[dir], filepath.Base(f))
		}

		// Print grouped files
		for dir, files := range dirMap {
			fmt.Printf("\n  %s/\n", dir)
			for _, file := range files {
				fmt.Printf("    - %s\n", file)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)

	// Add subcommands
	templateCmd.AddCommand(templateAddCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateRemoveCmd)
	templateCmd.AddCommand(templateShowCmd)

	// Flags for add command
	templateAddCmd.Flags().StringP("description", "d", "", "Description of the template")
	templateAddCmd.Flags().StringP("language", "l", "", "Override detected language/framework tag (e.g., React, Vue)")
	// Flags for show command
	templateShowCmd.Flags().Bool("files-only", false, "Only print the file list")
	templateShowCmd.Flags().Bool("summary", false, "Only print template metadata (no files)")
	templateShowCmd.Flags().Bool("json", false, "Output template details in JSON format")
	templateRemoveCmd.Flags().Bool("force", false, "Remove even if this template is set as default for a language")

	// Flags for list command
	templateListCmd.Flags().String("sort", "name", "Sort templates by: name or language")
	templateListCmd.Flags().Bool("quiet", false, "Only print template names (one per line)")
}
