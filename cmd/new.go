/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/kajvans/foundry/internal/config"
	"github.com/spf13/cobra"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new <project-name>",
	Short: "Create a new project from a template",
	Long: `Create a new project from a saved template. 

If you specify a language, Foundry will use the default template for that language.
If you specify a template name directly, it will use that template.
If neither is specified, Foundry will prompt you to choose.

The command will:
  - Copy the template files to a new directory
  - Replace placeholders like {{PROJECT_NAME}} and {{AUTHOR}}
  - Initialize git repository (optional)
`,
	Example: `  # Use the default Go template
	foundry new my-api --language Go

	# Use a specific saved template
	foundry new my-app --template react-starter

	# Choose target path explicitly
	foundry new my-project --language Python --path ~/projects

	# If neither language nor template is provided, Foundry lists options
	foundry new my-cli`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		language, _ := cmd.Flags().GetString("language")
		templateName, _ := cmd.Flags().GetString("template")
		targetPath, _ := cmd.Flags().GetString("path")
		noGit, _ := cmd.Flags().GetBool("no-git")
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
		varsKV, _ := cmd.Flags().GetStringArray("var")

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Determine which template to use
		var tmpl *config.Template

		if templateName != "" {
			// User specified template directly
			tmpl, err = config.GetTemplate(templateName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else if language != "" {
			// User specified language, use default template for that language
			defaultTmpl, err := config.GetLanguageDefault(language)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if defaultTmpl == "" {
				fmt.Fprintf(os.Stderr, "No default template set for language '%s'\n", language)
				fmt.Fprintf(os.Stderr, "Set one with: foundry config %s <template-name>\n", language)
				fmt.Fprintf(os.Stderr, "Or use --template to specify a template directly\n")
				os.Exit(1)
			}
			tmpl, err = config.GetTemplate(defaultTmpl)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			// No language or template specified, show available options
			templates, err := config.ListTemplates()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if len(templates) == 0 {
				fmt.Fprintf(os.Stderr, "No templates available. Add one with: foundry template add <name> <path>\n")
				os.Exit(1)
			}
			// Interactive selection only if allowed
			cfgInteractive := cfg.Interactive
			if !nonInteractive && cfgInteractive {
				// Step 1: choose language (arrow-key menu)
				langSet := map[string]struct{}{}
				for _, t := range templates {
					if t.Language != "" {
						langSet[t.Language] = struct{}{}
					}
				}
				langs := make([]string, 0, len(langSet))
				for l := range langSet {
					langs = append(langs, l)
				}
				sort.Strings(langs)

				if len(langs) == 0 {
					fmt.Fprintf(os.Stderr, "No languages detected from templates\n")
					os.Exit(1)
				}

				pageSize := 10
				if len(langs) < pageSize {
					pageSize = len(langs)
				}
				var chosenLang string
				if err := survey.AskOne(&survey.Select{
					Message:  "Select a language:",
					Options:  langs,
					PageSize: pageSize,
				}, &chosenLang); err != nil {
					fmt.Fprintf(os.Stderr, "Selection cancelled\n")
					os.Exit(1)
				}

				// Step 2: choose template within chosen language
				var filtered []config.Template
				for _, t := range templates {
					if t.Language == chosenLang {
						filtered = append(filtered, t)
					}
				}
				if len(filtered) == 0 {
					fmt.Fprintf(os.Stderr, "No templates available for language '%s'\n", chosenLang)
					os.Exit(1)
				}

				// Build display labels and prompt
				labels := make([]string, 0, len(filtered))
				for _, t := range filtered {
					label := t.Name
					if len(config.IsDefaultTemplate(t.Name)) > 0 {
						label = fmt.Sprintf("%s (default)", t.Name)
					}
					labels = append(labels, label)
				}
				pageSize = 10
				if len(labels) < pageSize {
					pageSize = len(labels)
				}
				var selectedLabel string
				if err := survey.AskOne(&survey.Select{
					Message:  fmt.Sprintf("Select a %s template:", chosenLang),
					Options:  labels,
					PageSize: pageSize,
				}, &selectedLabel); err != nil {
					fmt.Fprintf(os.Stderr, "Selection cancelled\n")
					os.Exit(1)
				}
				// Map back to template by stripping " (default)" suffix if present
				baseName := selectedLabel
				if idx := strings.Index(selectedLabel, " (default)"); idx >= 0 {
					baseName = selectedLabel[:idx]
				}
				var selected config.Template
				for _, t := range filtered {
					if t.Name == baseName {
						selected = t
						break
					}
				}
				tmpl, err = config.GetTemplate(selected.Name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Println("Available templates:")
				for i, t := range templates {
					defaults := config.IsDefaultTemplate(t.Name)
					defaultInfo := ""
					if len(defaults) > 0 {
						defaultInfo = fmt.Sprintf(" (default for: %v)", defaults)
					}
					fmt.Printf("  %d. %s - %s%s\n", i+1, t.Name, t.Language, defaultInfo)
				}
				fmt.Fprintf(os.Stderr, "\nPlease specify --language or --template (or enable interactive mode)\n")
				os.Exit(1)
			}
		}

		// Verify template path exists
		if _, err := os.Stat(tmpl.Path); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Template path no longer exists: %s\n", tmpl.Path)
			os.Exit(1)
		}

		// Determine target directory
		projectDir := projectName
		if targetPath != "" {
			projectDir = filepath.Join(targetPath, projectName)
		}

		// Check if target directory already exists
		if _, err := os.Stat(projectDir); err == nil {
			fmt.Fprintf(os.Stderr, "Error: Directory '%s' already exists\n", projectDir)
			os.Exit(1)
		}

		// Create project
		color.Cyan("Creating project '%s' from template '%s'...", projectName, tmpl.Name)
		fmt.Printf("  Language: %s\n", tmpl.Language)
		fmt.Printf("  Target: %s\n", projectDir)

		// Parse additional variables from --var key=value
		extraVars, err := parseVars(varsKV)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing --var: %v\n", err)
			os.Exit(1)
		}

		if err := createProject(tmpl, projectName, projectDir, cfg.Author, extraVars); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating project: %v\n", err)
			os.Exit(1)
		}

		color.Green("\n✓ Project '%s' created successfully!", projectName)
		fmt.Printf("  Location: %s\n", projectDir)

		// Initialize git if requested
		if !noGit {
			color.Magenta("\nInitializing git repository...")
			// TODO: Add git init logic
			color.HiBlack("  (git init not yet implemented)")
		}

		color.New(color.Bold).Println("\nNext steps:")
		fmt.Printf("  cd %s\n", projectName)
		if tmpl.Language == "Go" {
			fmt.Printf("  go mod tidy\n")
			fmt.Printf("  go build\n")
		} else if tmpl.Language == "JavaScript" || tmpl.Language == "TypeScript" || tmpl.Language == "React" {
			fmt.Printf("  npm install\n")
			fmt.Printf("  npm run dev\n")
		} else if tmpl.Language == "Python" {
			fmt.Printf("  pip install -r requirements.txt\n")
			fmt.Printf("  python main.py\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringP("language", "l", "", "Language/framework to use (uses default template for that language)")
	newCmd.Flags().StringP("template", "t", "", "Specific template to use")
	newCmd.Flags().StringP("path", "p", "", "Target path for the new project (default: current directory)")
	newCmd.Flags().Bool("no-git", false, "Skip git initialization")
	newCmd.Flags().Bool("non-interactive", false, "Do not prompt; require --language or --template")
	newCmd.Flags().StringArray("var", []string{}, "Template variable in key=value form (repeatable)")
}

// createProject copies the template to the target directory with placeholder replacement
func createProject(tmpl *config.Template, projectName, targetDir, author string, extraVars map[string]string) error {
	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Get absolute paths to avoid issues
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute target path: %w", err)
	}
	absSourceDir, err := filepath.Abs(tmpl.Path)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}

	// Resolve symlinks/junctions
	if realTarget, err := filepath.EvalSymlinks(absTargetDir); err == nil {
		absTargetDir = realTarget
	}
	if realSource, err := filepath.EvalSymlinks(absSourceDir); err == nil {
		absSourceDir = realSource
	}

	// Determine if target is inside source tree (copying within same repo)
	relTarget, relErr := filepath.Rel(absSourceDir, absTargetDir)
	targetInsideSource := (relErr == nil && !strings.HasPrefix(relTarget, ".."))

	// Load ignore patterns from .foundryignore in template root
	ignores := loadIgnorePatternsLocal(absSourceDir)

	// Copy files from template
	err = filepath.Walk(tmpl.Path, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip common bulky directories
		if info.IsDir() {
			base := info.Name()
			if base == "node_modules" || base == "vendor" || base == ".venv" || base == "dist" || base == "build" {
				return filepath.SkipDir
			}
		}

		// Skip the target directory (and its children) if it's inside the template path (prevents infinite loop)
		if targetInsideSource {
			relSrcFromSource, _ := filepath.Rel(absSourceDir, srcPath)
			if relSrcFromSource == relTarget || strings.HasPrefix(relSrcFromSource+string(os.PathSeparator), relTarget+string(os.PathSeparator)) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Calculate relative path
		relPath, err := filepath.Rel(tmpl.Path, srcPath)
		if err != nil {
			return err
		}

		// Apply .foundryignore patterns
		if matchIgnoreLocal(filepath.ToSlash(relPath), ignores) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Determine destination path
		dstPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file with placeholder replacement
		return copyFileWithReplacements(srcPath, dstPath, projectName, author, info.Mode(), extraVars)
	})

	return err
}

// copyFileWithReplacements copies a file and replaces placeholders
func copyFileWithReplacements(src, dst, projectName, author string, mode os.FileMode, extraVars map[string]string) error {
	// Read source file
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}

	// Binary detection: if the first chunk contains NUL, treat as binary and skip replacements
	if looksBinary(content) {
		return os.WriteFile(dst, content, mode)
	}

	// Replace placeholders
	contentStr := string(content)
	contentStr = strings.ReplaceAll(contentStr, "{{PROJECT_NAME}}", projectName)
	contentStr = strings.ReplaceAll(contentStr, "{{AUTHOR}}", author)
	contentStr = strings.ReplaceAll(contentStr, "{{PROJECT_NAME_LOWER}}", strings.ToLower(projectName))
	contentStr = strings.ReplaceAll(contentStr, "{{PROJECT_NAME_UPPER}}", strings.ToUpper(projectName))

	// Extra variables: replace {{KEY}} with provided values (case-sensitive)
	for k, v := range extraVars {
		placeholder := "{{" + k + "}}"
		contentStr = strings.ReplaceAll(contentStr, placeholder, v)
	}

	// Write to destination
	if err := os.WriteFile(dst, []byte(contentStr), mode); err != nil {
		return fmt.Errorf("failed to write %s: %w", dst, err)
	}

	return nil
}

// parseVars parses --var key=value entries into a map
func parseVars(kvs []string) (map[string]string, error) {
	m := map[string]string{}
	for _, kv := range kvs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid var, expected key=value")
		}
		key := strings.TrimSpace(parts[0])
		val := parts[1]
		if key == "" {
			return nil, errors.New("variable key cannot be empty")
		}
		m[key] = val
	}
	return m, nil
}

// looksBinary reports whether data likely represents a binary file
func looksBinary(data []byte) bool {
	n := len(data)
	if n > 8000 {
		n = 8000
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// Local ignore utilities for copy operation (reads template's .foundryignore)
func loadIgnorePatternsLocal(root string) []string {
	f, err := os.Open(filepath.Join(root, ".foundryignore"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var patterns []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

func matchIgnoreLocal(rel string, patterns []string) bool {
	r := filepath.ToSlash(rel)
	for _, p := range patterns {
		pp := filepath.ToSlash(strings.TrimSuffix(p, "/"))
		// Glob-like match using simple prefix or exact match
		if ok, _ := filepath.Match(pp, r); ok {
			return true
		}
		if strings.HasPrefix(r+"/", pp+"/") {
			return true
		}
	}
	return false
}
