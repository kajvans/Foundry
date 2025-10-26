package detect

import (
	"fmt"
	"os/exec"
	"sort"

	"github.com/kajvans/foundry/internal/config"
)

type ScanResult struct {
	Languages      map[string]bool
	PackageManagers map[string]bool
	DevTools       map[string]bool
}

// ScanSystem does all the logic of checking binaries
func ScanSystem() *ScanResult {
	categories := map[string]map[string]string{
		"Languages": {
			"Go":      "go",
			"Python":  "python3",
			"Node.js": "node",
			"Rust":    "rustc",
			"Java":    "javac",
			"C++":     "g++",
			"PHP":     "php",
		},
		"Package Managers": {
			"pip":      "pip3",
			"npm":      "npm",
			"yarn":     "yarn",
			"pnpm":     "pnpm",
			"cargo":    "cargo",
			"maven":    "mvn",
			"gradle":   "gradle",
			"composer": "composer",
			"make":     "make",
			"cmake":    "cmake",
		},
		"Development Tools": {
			"git":    "git",
			"docker": "docker",
		},
	}

	result := &ScanResult{
		Languages:      map[string]bool{},
		PackageManagers: map[string]bool{},
		DevTools:       map[string]bool{},
	}

	for category, tools := range categories {
		for name, bin := range tools {
			found := false
			if _, err := exec.LookPath(bin); err == nil {
				found = true
			}
			switch category {
			case "Languages":
				result.Languages[name] = found
			case "Package Managers":
				result.PackageManagers[name] = found
			case "Development Tools":
				result.DevTools[name] = found
			}
		}
	}

	return result
}

// PrintResult prints the detected tools nicely
func PrintResult(result *ScanResult) {
	categories := map[string]map[string]bool{
		"Languages":      result.Languages,
		"Package Managers": result.PackageManagers,
		"Development Tools": result.DevTools,
	}

	for category, tools := range categories {
		fmt.Printf("=== %s ===\n", category)
		names := make([]string, 0, len(tools))
		for name := range tools {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if tools[name] {
				fmt.Printf("✅ %-10s\n", name)
			} else {
				fmt.Printf("❌ %-10s\n", name)
			}
		}
		fmt.Println()
	}
}

func SaveConfig(ScanResult *ScanResult) error {
	// Convert maps to slices
	installedLanguages := []string{}
	for lang, found := range ScanResult.Languages {
		if found {
			installedLanguages = append(installedLanguages, lang)
		}
	}

	installedPackageManagers := []string{}
	for pm, found := range ScanResult.PackageManagers {
		if found {
			installedPackageManagers = append(installedPackageManagers, pm)
		}
	}

	installedDevTools := []string{}
	for tool, found := range ScanResult.DevTools {
		if found {
			installedDevTools = append(installedDevTools, tool)
		}
	}

	// Save to config
	if err := config.SetConfigValue("installed_languages", installedLanguages); err != nil {
		return err
	}
	if err := config.SetConfigValue("installed_package_managers", installedPackageManagers); err != nil {
		return err
	}
	if err := config.SetConfigValue("installed_dev_tools", installedDevTools); err != nil {
		return err
	}

	return nil
}