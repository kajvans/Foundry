package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Template represents a saved project template
type Template struct {
	Name        string   `yaml:"name"`
	Path        string   `yaml:"path"`
	Language    string   `yaml:"language"`
	Description string   `yaml:"description"`
	Files       []string `yaml:"files,omitempty"`
}

type Config struct {
	Author          string `yaml:"author"`
	License         string `yaml:"license"`
	DefaultLanguage string `yaml:"default_language"`
	Docker          bool   `yaml:"docker"`
	Interactive     bool   `yaml:"interactive"`

	// Detected tools on the system
	InstalledLanguages       []string `yaml:"installed_languages"`
	InstalledPackageManagers []string `yaml:"installed_package_managers"`
	InstalledDevTools        []string `yaml:"installed_dev_tools"`
	VSCodePath               string   `yaml:"vscode_path,omitempty"`

	// Saved templates
	Templates []Template `yaml:"templates,omitempty"`

	// Default templates per language (e.g., "Go": "my-go-template")
	LanguageDefaults map[string]string `yaml:"language_defaults,omitempty"`
}

// configPathOverride allows overriding the default config file path.
// When set (non-empty), getConfigPath will return this path instead of ~/.foundry/config.yaml
var configPathOverride string

// SetConfigPathOverride sets a custom config file path (absolute or relative).
// If empty, the default path (~/.foundry/config.yaml) will be used.
func SetConfigPathOverride(p string) {
	configPathOverride = p
}

func InitConfig() {
	// Ensure config directory exists on initialization
	_, err := getConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	//create a default config if none exists
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		return
	}
	if cfg == nil {
		defaultCfg := &Config{
			Author:                   "Your Name",
			License:                  "MIT",
			DefaultLanguage:          "",
			Docker:                   false,
			Interactive:              true,
			InstalledLanguages:       []string{},
			InstalledPackageManagers: []string{},
			InstalledDevTools:        []string{},
			Templates:                []Template{},
			LanguageDefaults:         make(map[string]string),
			VSCodePath:               "",
		}
		if err := SaveConfig(defaultCfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}
	// Ensure config file exists
	configPath, err := getConfigPath()
	if err == nil {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if err := SaveConfig(&Config{}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}
}

// getConfigPath returns the full path to the config file depending on OS
func getConfigPath() (string, error) {
	if configPathOverride != "" {
		// If user provided a relative path, make it absolute relative to cwd
		if !filepath.IsAbs(configPathOverride) {
			if abs, err := filepath.Abs(configPathOverride); err == nil {
				return abs, nil
			}
		}
		return configPathOverride, nil
	}
	var home string
	if h, err := os.UserHomeDir(); err == nil {
		home = h
	} else {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	configDir := filepath.Join(home, ".foundry")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// LoadConfig reads the config file from disk, or returns default if missing
func LoadConfig() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Author:                   "",
		License:                  "MIT",
		DefaultLanguage:          "",
		Docker:                   false,
		Interactive:              true,
		InstalledLanguages:       []string{},
		InstalledPackageManagers: []string{},
		InstalledDevTools:        []string{},
		Templates:                []Template{},
		LanguageDefaults:         make(map[string]string),
		VSCodePath:               "",
	}

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		// file doesn't exist, return default
		return cfg, nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// SaveConfig writes the config to disk
func SaveConfig(cfg *Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create config file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func SetConfigValue(key string, value interface{}) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	switch key {
	case "author":
		if v, ok := value.(string); ok {
			cfg.Author = v
		}
	case "license":
		if v, ok := value.(string); ok {
			cfg.License = v
		}
	case "default_language":
		if v, ok := value.(string); ok {
			cfg.DefaultLanguage = v
		}
	case "docker":
		if v, ok := value.(bool); ok {
			cfg.Docker = v
		}
	case "interactive":
		if v, ok := value.(bool); ok {
			cfg.Interactive = v
		}
	case "installed_languages":
		if v, ok := value.([]string); ok {
			cfg.InstalledLanguages = v
		}
	case "installed_package_managers":
		if v, ok := value.([]string); ok {
			cfg.InstalledPackageManagers = v
		}
	case "installed_dev_tools":
		if v, ok := value.([]string); ok {
			cfg.InstalledDevTools = v
		}
	case "vscode_path":
		if v, ok := value.(string); ok {
			cfg.VSCodePath = v
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return SaveConfig(cfg)
}

func GetConfigValue(key string) (interface{}, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	switch key {
	case "author":
		return cfg.Author, nil
	case "license":
		return cfg.License, nil
	case "default_language":
		return cfg.DefaultLanguage, nil
	case "docker":
		return cfg.Docker, nil
	case "interactive":
		return cfg.Interactive, nil
	case "installed_languages":
		return cfg.InstalledLanguages, nil
	case "installed_package_managers":
		return cfg.InstalledPackageManagers, nil
	case "installed_dev_tools":
		return cfg.InstalledDevTools, nil
	case "git":
		//check if git is inside installed dev tools
		for _, tool := range cfg.InstalledDevTools {
			if tool == "git" {
				return true, nil
			}
		}
		return false, nil
	case "vscode_path":
		return cfg.VSCodePath, nil
	default:
		return nil, fmt.Errorf("unknown config key: %s", key)
	}
}

func PrintConfig() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Author: %s\n", cfg.Author)
	fmt.Printf("License: %s\n", cfg.License)
	fmt.Printf("Default Language: %s\n", cfg.DefaultLanguage)
	fmt.Printf("Docker: %t\n", cfg.Docker)
	fmt.Printf("Interactive: %t\n", cfg.Interactive)
	fmt.Printf("Installed Languages: %v\n", cfg.InstalledLanguages)
	fmt.Printf("Installed Package Managers: %v\n", cfg.InstalledPackageManagers)
	fmt.Printf("Installed Dev Tools: %v\n", cfg.InstalledDevTools)
	fmt.Printf("Templates: %d saved\n", len(cfg.Templates))

	// Show language defaults if any are set
	if len(cfg.LanguageDefaults) > 0 {
		fmt.Printf("\nLanguage Defaults:\n")
		for lang, tmpl := range cfg.LanguageDefaults {
			fmt.Printf("  %s: %s\n", lang, tmpl)
		}
	}
}

// AddTemplate adds a new template to the config
func AddTemplate(tmpl Template) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	// Check if template with same name already exists
	for i, t := range cfg.Templates {
		if t.Name == tmpl.Name {
			// Replace existing template
			cfg.Templates[i] = tmpl
			return SaveConfig(cfg)
		}
	}

	// Add new template
	cfg.Templates = append(cfg.Templates, tmpl)
	return SaveConfig(cfg)
}

// RemoveTemplate removes a template by name
func RemoveTemplate(name string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	found := false
	newTemplates := []Template{}
	for _, t := range cfg.Templates {
		if t.Name != name {
			newTemplates = append(newTemplates, t)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("template '%s' not found", name)
	}

	cfg.Templates = newTemplates
	return SaveConfig(cfg)
}

// GetTemplate retrieves a template by name
func GetTemplate(name string) (*Template, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	for _, t := range cfg.Templates {
		if t.Name == name {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("template '%s' not found", name)
}

// ListTemplates returns all saved templates
func ListTemplates() ([]Template, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Templates, nil
}

// SetLanguageDefault sets the default template for a specific language
func SetLanguageDefault(language, templateName string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	// Verify template exists
	found := false
	for _, t := range cfg.Templates {
		if t.Name == templateName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("template '%s' not found", templateName)
	}

	// Initialize map if nil
	if cfg.LanguageDefaults == nil {
		cfg.LanguageDefaults = make(map[string]string)
	}

	cfg.LanguageDefaults[language] = templateName
	return SaveConfig(cfg)
}

// GetLanguageDefault returns the default template for a specific language
func GetLanguageDefault(language string) (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}

	if cfg.LanguageDefaults == nil {
		return "", nil
	}

	return cfg.LanguageDefaults[language], nil
}

// ClearLanguageDefault removes the default template for a specific language
func ClearLanguageDefault(language string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	if cfg.LanguageDefaults != nil {
		delete(cfg.LanguageDefaults, language)
	}

	return SaveConfig(cfg)
}

// IsDefaultTemplate checks if a template is set as default for any language
func IsDefaultTemplate(templateName string) []string {
	cfg, err := LoadConfig()
	if err != nil {
		return []string{}
	}

	languages := []string{}
	if cfg.LanguageDefaults != nil {
		for lang, tmpl := range cfg.LanguageDefaults {
			if tmpl == templateName {
				languages = append(languages, lang)
			}
		}
	}
	return languages
}
