package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/kajvans/foundry/internal/config"
	"github.com/kajvans/foundry/internal/post"
	"github.com/kajvans/foundry/internal/project"
	"github.com/kajvans/foundry/internal/utils"
	"github.com/spf13/cobra"
)

const (
	maxBinaryCheckBytes = 8000
	defaultPageSize     = 10
)

var ignoredDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".venv":        true,
	"dist":         true,
	"build":        true,
	".git":         true,
}

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

	# Fetch template from a Git repository
	foundry new my-project --git https://github.com/user/template-repo

	# Choose target path explicitly
	foundry new my-project --language Python --path ~/projects

	# If neither language nor template is provided, Foundry lists options
	foundry new my-cli`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		language, _ := cmd.Flags().GetString("language")
		templateName, _ := cmd.Flags().GetString("template")
		gitURL, _ := cmd.Flags().GetString("git")
		targetPath, _ := cmd.Flags().GetString("path")
		noGit, _ := cmd.Flags().GetBool("no-git")
		noPost, _ := cmd.Flags().GetBool("no-post")
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
		varsKV, _ := cmd.Flags().GetStringArray("var")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		cfg, err := config.LoadConfig()
		if err != nil {
			exitWithError("Error loading config: %v", err)
		}

		//check if git exists
		gitExists, err := config.GetConfigValue("git")

		if gitURL != "" && gitExists.(bool) {
			projectDir := determineProjectDir(projectName, targetPath)

			// Check early if the directory already exists
			if _, err := os.Stat(projectDir); err == nil {
				exitWithError("Directory '%s' already exists", projectDir)
			}

			// Clone repository
			cmd := exec.Command("git", "clone", gitURL, projectDir)
			if err := cmd.Run(); err != nil {
				exitWithError("Failed to clone git repository: %v", err)
			}
		} else {
			// Determine which template to use
			tmpl := selectTemplate(cfg, templateName, language, nonInteractive)

			// Verify template path exists
			if _, err := os.Stat(tmpl.Path); os.IsNotExist(err) {
				exitWithError("Template path no longer exists: %s", tmpl.Path)
			}

			projectDir := determineProjectDir(projectName, targetPath)

			// Check if target directory already exists
			if _, err := os.Stat(projectDir); err == nil {
				exitWithError("Directory '%s' already exists", projectDir)
			}

			// Parse additional variables
			extraVars, err := utils.ParseVars(varsKV)
			if err != nil {
				exitWithError("Error parsing --var: %v", err)
			}

			// Create or preview project
			printProjectInfo(projectName, tmpl, projectDir)
			if dryRun {
				summary, err := project.PreviewFromTemplate(tmpl, projectName, projectDir, cfg.Author, extraVars)
				if err != nil {
					exitWithError("Error previewing project: %v", err)
				}
				color.Yellow("\nDry run: no files written, no git init.")
				fmt.Printf("  Would create %d files:\n", len(summary.Files))
				// show up to 20 entries
				maxShow := 20
				if len(summary.Files) < maxShow {
					maxShow = len(summary.Files)
				}
				for i := 0; i < maxShow; i++ {
					fmt.Printf("    - %s\n", summary.Files[i])
				}
				if len(summary.Files) > maxShow {
					fmt.Printf("    ... and %d more\n", len(summary.Files)-maxShow)
				}
				return
			}
			if err := project.CreateFromTemplate(tmpl, projectName, projectDir, cfg.Author, extraVars); err != nil {
				exitWithError("Error creating project: %v", err)
			}

			// Run post-create language-specific steps unless disabled or dry-run
			if !dryRun {
				if !noPost {
					color.Magenta("\nRunning language-specific setup...")
					if err := post.RunLanguagePost(tmpl.Language, projectDir); err != nil {
						color.Yellow("⚠ Post-create steps failed: %v", err)
					} else {
						color.Green("✓ Post-create steps finished.")
					}
				} else {
					color.Yellow("\n⚠ Post-create steps skipped as per --no-post flag.")
				}
			}

			printSuccessMessage(projectName, projectDir, tmpl.Language, noGit, noPost)
		}

	},
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringP("language", "l", "", "Language/framework to use (uses default template for that language)")
	newCmd.Flags().StringP("template", "t", "", "Specific template to use")
	newCmd.Flags().StringP("git", "g", "", "Git repository URL to fetch template from (e.g., https://github.com/user/repo)")
	newCmd.Flags().StringP("path", "p", "", "Target path for the new project (default: current directory)")
	newCmd.Flags().Bool("no-git", false, "Skip git initialization")
	newCmd.Flags().Bool("no-post", false, "Skip language-specific post-create commands (npm/pip/go)")
	newCmd.Flags().Bool("non-interactive", false, "Do not prompt; require --language or --template")
	newCmd.Flags().StringArray("var", []string{}, "Template variable in key=value form (repeatable)")
	newCmd.Flags().Bool("dry-run", false, "Preview actions without writing files or initializing git")
}

// exitWithError prints error and exits with code 1
func exitWithError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// selectTemplate determines which template to use based on flags and interactive mode
func selectTemplate(cfg *config.Config, templateName, language string, nonInteractive bool) *config.Template {
	if templateName != "" {
		return selectByName(templateName)
	}
	if language != "" {
		return selectByLanguage(language)
	}
	return selectInteractively(cfg, nonInteractive)
}

// selectByName gets template by explicit name
func selectByName(name string) *config.Template {
	tmpl, err := config.GetTemplate(name)
	if err != nil {
		exitWithError("%v", err)
	}
	return tmpl
}

// selectByLanguage gets default template for a language
func selectByLanguage(language string) *config.Template {
	defaultTmpl, err := config.GetLanguageDefault(language)
	if err != nil {
		exitWithError("%v", err)
	}
	if defaultTmpl == "" {
		exitWithError("No default template set for language '%s'\nSet one with: foundry config %s <template-name>\nOr use --template to specify a template directly", language, language)
	}
	tmpl, err := config.GetTemplate(defaultTmpl)
	if err != nil {
		exitWithError("%v", err)
	}
	return tmpl
}

// selectInteractively shows template selection UI or lists available templates
func selectInteractively(cfg *config.Config, nonInteractive bool) *config.Template {
	templates, err := config.ListTemplates()
	if err != nil {
		exitWithError("%v", err)
	}
	if len(templates) == 0 {
		exitWithError("No templates available. Add one with: foundry template add <name> <path>")
	}

	if nonInteractive || !cfg.Interactive {
		listTemplatesAndExit(templates)
	}

	// Interactive mode: two-step selection
	chosenLang := selectLanguage(templates)
	return selectTemplateForLanguage(templates, chosenLang)
}

// selectLanguage shows language selection menu
func selectLanguage(templates []config.Template) string {
	langSet := make(map[string]struct{})
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
		exitWithError("No languages detected from templates")
	}

	pageSize := utils.Min(len(langs), defaultPageSize)
	var chosenLang string
	if err := survey.AskOne(&survey.Select{
		Message:  "Select a language:",
		Options:  langs,
		PageSize: pageSize,
	}, &chosenLang); err != nil {
		exitWithError("Selection cancelled")
	}
	return chosenLang
}

// selectTemplateForLanguage shows template selection menu for chosen language
func selectTemplateForLanguage(templates []config.Template, language string) *config.Template {
	var filtered []config.Template
	for _, t := range templates {
		if t.Language == language {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		exitWithError("No templates available for language '%s'", language)
	}

	labels := make([]string, 0, len(filtered))
	for _, t := range filtered {
		label := t.Name
		if len(config.IsDefaultTemplate(t.Name)) > 0 {
			label = fmt.Sprintf("%s (default)", t.Name)
		}
		labels = append(labels, label)
	}

	pageSize := utils.Min(len(labels), defaultPageSize)
	var selectedLabel string
	if err := survey.AskOne(&survey.Select{
		Message:  fmt.Sprintf("Select a %s template:", language),
		Options:  labels,
		PageSize: pageSize,
	}, &selectedLabel); err != nil {
		exitWithError("Selection cancelled")
	}

	// Strip " (default)" suffix
	baseName := strings.TrimSuffix(selectedLabel, " (default)")
	for _, t := range filtered {
		if t.Name == baseName {
			tmpl, err := config.GetTemplate(t.Name)
			if err != nil {
				exitWithError("%v", err)
			}
			return tmpl
		}
	}
	exitWithError("Template not found")
	return nil
}

// listTemplatesAndExit lists all templates and exits
func listTemplatesAndExit(templates []config.Template) {
	fmt.Println("Available templates:")
	for i, t := range templates {
		defaults := config.IsDefaultTemplate(t.Name)
		defaultInfo := ""
		if len(defaults) > 0 {
			defaultInfo = fmt.Sprintf(" (default for: %v)", defaults)
		}
		fmt.Printf("  %d. %s - %s%s\n", i+1, t.Name, t.Language, defaultInfo)
	}
	exitWithError("Please specify --language or --template (or enable interactive mode)")
}

// determineProjectDir calculates the target directory for the project
func determineProjectDir(projectName, targetPath string) string {
	if targetPath != "" {
		return filepath.Join(targetPath, projectName)
	}
	return projectName
}

// printProjectInfo displays project creation details
func printProjectInfo(projectName string, tmpl *config.Template, projectDir string) {
	color.Cyan("Creating project '%s' from template '%s'...", projectName, tmpl.Name)
	fmt.Printf("  Language: %s\n", tmpl.Language)
	fmt.Printf("  Target: %s\n", projectDir)
}

// printSuccessMessage displays success message and next steps
func printSuccessMessage(projectName, projectDir, language string, noGit bool, noPost bool) {
	color.Green("\n✓ Project '%s' created successfully!", projectName)
	fmt.Printf("  Location: %s\n", projectDir)

	// Setup git repository
	setupGitRepo(projectDir, noGit, language)

	//TODO: Add code here to open project in VS Code if available
	vscodePath, err := config.GetConfigValue("vscode_path")
	if err == nil {
		if pathStr, ok := vscodePath.(string); ok && pathStr != "" {
			color.Magenta("\nOpening project in VS Code...")
			cmd := exec.Command(pathStr, projectDir)
			if err := cmd.Start(); err != nil {
				color.Red("✗ Failed to open VS Code: %v", err)
			} else {
				color.Green("✓ VS Code opened.")
			}
		}
	}

	//printLanguageSpecificSteps(language)
	color.New(color.Bold).Println("\nNext steps:")
	fmt.Printf("  cd %s\n", projectName)
	if(!noPost){
		fmt.Printf("  Run the following commands to get started with your %s project:\n", language)
		printLanguageSpecificSteps(language)
	}
}

func setupGitRepo(projectDir string, noGit bool, language string) error {

	if !noGit {
		color.Magenta("\nInitializing git repository...")
		cmd := exec.Command("git", "init", projectDir)
		if err := cmd.Run(); err != nil {
			color.Red("✗ Failed to initialize git repository: %v", err)
		} else {
			color.Green("✓ Git repository initialized.")
		}

		//check if gitignore exists in folder
		if _, err := os.Stat(filepath.Join(projectDir, ".gitignore")); os.IsNotExist(err) {
			//download default gitignore for language
			color.Magenta("Adding default .gitignore for %s...", language)
			gitignoreContent := getDefaultGitignore(language)
			if gitignoreContent != "" {
				gitignorePath := filepath.Join(projectDir, ".gitignore")
				if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
					color.Red("✗ Failed to create .gitignore: %v", err)
				} else {
					color.Green("✓ .gitignore created.")
				}
			} else {
				color.Yellow("⚠ No default .gitignore available for %s", language)
			}
		}

		// 3. Run: git add .

		cmd = exec.Command("git", "-C", projectDir, "add", ".")
		if err := cmd.Run(); err != nil {
			color.Red("✗ Failed to add files to git: %v", err)
		} else {
			color.Green("✓ Files added to git.")
		}

		// 4. Run: git commit -m "Initial commit from Foundry"
		cmd = exec.Command("git", "-C", projectDir, "commit", "-m", "Initial commit from Foundry")
		if err := cmd.Run(); err != nil {
			color.Red("✗ Failed to commit files to git: %v", err)
		} else {
			color.Green("✓ Initial commit created.")
		}

	} else {
		color.Yellow("\n⚠ Git initialization skipped as per --no-git flag.")
	}
	return nil
}

func getDefaultGitignore(language string) string {
	//download from this link https://raw.githubusercontent.com/github/gitignore/refs/heads/main/$language.gitignore
	//make first letter uppercase and rest lowercase
	langFormatted := utils.CapitalizeFirst(language)
	url := fmt.Sprintf("https://raw.githubusercontent.com/github/gitignore/refs/heads/main/%s.gitignore", langFormatted)

	resp, err := exec.Command("curl", "-sL", url).Output()
	if err != nil {
		return ""
	}
	return string(resp)
}

// printLanguageSpecificSteps shows commands for specific language
func printLanguageSpecificSteps(language string) {
	switch language {
	case "Go":
		fmt.Println("  go mod tidy")
		fmt.Println("  go build")
	case "JavaScript", "TypeScript", "React":
		fmt.Println("  npm install")
		fmt.Println("  npm run dev")
	case "Python":
		fmt.Println("  pip install -r requirements.txt")
		fmt.Println("  python main.py")
	case "Rust":
		fmt.Println("  cargo build")
		fmt.Println("  cargo run")
	}
}

// ...verplaatst: createProject -> internal/project.CreateFromTemplate

// copyFileWithReplacements copies a file and replaces placeholders
func copyFileWithReplacements(src, dst, projectName, author string, mode os.FileMode, extraVars map[string]string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}

	// Skip placeholder replacement for binary files
	if utils.IsBinary(content, maxBinaryCheckBytes) {
		return os.WriteFile(dst, content, mode)
	}

	// Replace placeholders
	contentStr := utils.ReplacePlaceholders(string(content), projectName, author, extraVars)

	return os.WriteFile(dst, []byte(contentStr), mode)
}
