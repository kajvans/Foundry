package template

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Template represents a saved project template
type Template struct {
	Name        string   `yaml:"name"`
	Path        string   `yaml:"path"`
	Language    string   `yaml:"language"`
	Description string   `yaml:"description"`
	Files       []string `yaml:"files,omitempty"` // List of files in template
}

// languageIndicators maps file extensions and filenames to languages
var languageIndicators = map[string]string{
	// Extensions
	".go":    "Go",
	".mod":   "Go",
	".py":    "Python",
	".js":    "JavaScript",
	".ts":    "TypeScript",
	".jsx":   "React",
	".tsx":   "React",
	".rs":    "Rust",
	".java":  "Java",
	".kt":    "Kotlin",
	".cpp":   "C++",
	".c":     "C",
	".cs":    "C#",
	".php":   "PHP",
	".rb":    "Ruby",
	".swift": "Swift",
	".vue":   "Vue",

	// Specific filenames
	"package.json":     "JavaScript",
	"tsconfig.json":    "TypeScript",
	"Cargo.toml":       "Rust",
	"pom.xml":          "Java",
	"build.gradle":     "Java",
	"Gemfile":          "Ruby",
	"composer.json":    "PHP",
	"requirements.txt": "Python",
	"Pipfile":          "Python",
	"go.mod":           "Go",
	"Makefile":         "C/C++",
}

// DetectLanguage scans a directory and determines the primary language
func DetectLanguage(dir string) (string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", dir)
	}

	languageCounts := make(map[string]int)

	// Load ignore patterns from root .foundryignore if present
	ignores := loadIgnorePatterns(dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip common directories
			base := filepath.Base(path)
			if base == "node_modules" || base == ".git" || base == "vendor" || base == "target" || base == "build" || base == "dist" {
				return filepath.SkipDir
			}
			// Skip ignored directories
			rel, _ := filepath.Rel(dir, path)
			if matchIgnore(rel, ignores) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip ignored files
		rel, _ := filepath.Rel(dir, path)
		if matchIgnore(rel, ignores) {
			return nil
		}

		// Check by filename first
		basename := filepath.Base(path)
		if lang, ok := languageIndicators[basename]; ok {
			languageCounts[lang] += 5 // Higher weight for specific files
			return nil
		}

		// Check by extension
		ext := filepath.Ext(path)
		if lang, ok := languageIndicators[ext]; ok {
			languageCounts[lang]++
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(languageCounts) == 0 {
		return "Unknown", nil
	}

	// Find the most common language
	maxCount := 0
	primaryLang := "Unknown"
	for lang, count := range languageCounts {
		if count > maxCount {
			maxCount = count
			primaryLang = lang
		}
	}

	return primaryLang, nil
}

// ScanTemplate scans a directory and creates a Template
func ScanTemplate(name, path, description string) (*Template, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template directory does not exist: %s", absPath)
	}

	lang, err := DetectLanguage(absPath)
	if err != nil {
		return nil, err
	}

	// List files in template
	ignores := loadIgnorePatterns(absPath)
	var files []string
	err = filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(absPath, p)
			if matchIgnore(relPath, ignores) {
				return nil
			}
			files = append(files, relPath)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list template files: %w", err)
	}

	tmpl := &Template{
		Name:        name,
		Path:        absPath,
		Language:    lang,
		Description: description,
		Files:       files,
	}

	return tmpl, nil
}

// ValidateName checks if a template name is valid
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return fmt.Errorf("template name contains invalid characters")
	}
	return nil
}

// loadIgnorePatterns reads .foundryignore in the root directory (if present)
// and returns a list of glob patterns relative to the root.
func loadIgnorePatterns(root string) []string {
	path := filepath.Join(root, ".foundryignore")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var patterns []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// matchIgnore checks if a relative path matches any of the ignore patterns.
// It supports simple glob matching via filepath.Match and prefix directory matching.
func matchIgnore(relPath string, patterns []string) bool {
	norm := filepath.ToSlash(relPath)
	for _, p := range patterns {
		pp := filepath.ToSlash(strings.TrimSuffix(p, "/"))
		// Direct glob match
		if ok, _ := filepath.Match(pp, norm); ok {
			return true
		}
		// Prefix directory match
		if strings.HasPrefix(norm+"/", pp+"/") {
			return true
		}
	}
	return false
}
