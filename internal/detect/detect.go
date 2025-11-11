package detect

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/kajvans/foundry/internal/config"
)

type ScanResult struct {
	Languages       map[string]bool
	PackageManagers map[string]bool
	DevTools        map[string]bool
	VSCodePath      string // Path to VS Code executable
}

// checkVSCode checks for VS Code installation on various platforms
// Returns the path to VS Code executable if found, empty string otherwise
func checkVSCode() string {
	// Try PATH first (works if user added code to PATH)
	if codePath, err := exec.LookPath("code"); err == nil {
		return codePath
	}

	// Windows-specific checks
	if runtime.GOOS == "windows" {
		// Check common installation paths (prefer code.cmd for CLI usage)
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			paths := []string{
				filepath.Join(userProfile, "AppData", "Local", "Programs", "Microsoft VS Code", "bin", "code.cmd"),
				filepath.Join(userProfile, "AppData", "Local", "Programs", "Microsoft VS Code", "Code.exe"),
			}
			for _, path := range paths {
				if _, err := os.Stat(path); err == nil {
					return path
				}
			}
		}

		// Check Program Files
		programFiles := os.Getenv("ProgramFiles")
		if programFiles != "" {
			paths := []string{
				filepath.Join(programFiles, "Microsoft VS Code", "bin", "code.cmd"),
				filepath.Join(programFiles, "Microsoft VS Code", "Code.exe"),
			}
			for _, path := range paths {
				if _, err := os.Stat(path); err == nil {
					return path
				}
			}
		}

		// Check for code.cmd in PATH
		if codePath, err := exec.LookPath("code.cmd"); err == nil {
			return codePath
		}

		// Check if VS Code is currently running (fallback for custom install locations)
		// Windows: check for Code.exe process and get its path
		cmd := exec.Command("powershell", "-Command", "Get-Process Code -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty Path")
		if output, err := cmd.Output(); err == nil && len(output) > 0 {
			path := strings.TrimSpace(string(output))
			if path != "" {
				// Try to find bin\code.cmd in the same directory structure
				dir := filepath.Dir(path)
				codeCmdPath := filepath.Join(dir, "bin", "code.cmd")
				if _, err := os.Stat(codeCmdPath); err == nil {
					return codeCmdPath
				}
				return path
			}
		}
	}

	// Linux-specific checks
	if runtime.GOOS == "linux" {
		// Check common Linux installation paths
		paths := []string{
			"/usr/bin/code",
			"/usr/local/bin/code",
			"/snap/bin/code",
			"/usr/share/code/code",
		}
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		// Check if VS Code is running (works for custom install locations)
		cmd := exec.Command("pgrep", "-x", "code")
		if err := cmd.Run(); err == nil {
			// Try to get the path from the running process
			cmd = exec.Command("sh", "-c", "readlink -f /proc/$(pgrep -x code | head -1)/exe")
			if output, err := cmd.Output(); err == nil {
				return strings.TrimSpace(string(output))
			}
		}
	}

	// macOS-specific check
	if runtime.GOOS == "darwin" {
		if _, err := os.Stat("/Applications/Visual Studio Code.app"); err == nil {
			return "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
		}
	}

	return ""
}

// ScanSystem does all the logic of checking binaries
func ScanSystem() *ScanResult {
	categories := map[string]map[string]string{
		"Languages": {
			"Go":         "go",
			"Python":     "python3",
			"Node.js":    "node",
			"Rust":       "rustc",
			"Java":       "javac",
			"C++":        "g++",
			"PHP":        "php",
			"Ruby":       "ruby",
			"Swift":      "swift",
			"Kotlin":     "kotlinc",
			"C#":         "csc",
			"C":          "gcc",
			"TypeScript": "tsc",
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
			"bundler":  "bundle",
			"brew":     "brew",
			"apt":      "apt",
		},
		"Development Tools": {
			"git":       "git",
			"docker":    "docker",
			"kubectl":   "kubectl",
			"apache":    "apache2",
			"nginx":     "nginx",
			"terraform": "terraform",
			"ansible":   "ansible",
			"sqlite3":   "sqlite3",
			"mysql":     "mysql",
			"psql":      "psql",
			"vscode":    "code",
		},
	}

	result := &ScanResult{
		Languages:       map[string]bool{},
		PackageManagers: map[string]bool{},
		DevTools:        map[string]bool{},
	}

	for category, tools := range categories {
		for name, bin := range tools {
			found := false

			// Special case for VS Code - use custom detection
			if name == "vscode" {
				vscodePath := checkVSCode()
				found = vscodePath != ""
				result.VSCodePath = vscodePath
			} else {
				if _, err := exec.LookPath(bin); err == nil {
					found = true
				}
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
		"Languages":         result.Languages,
		"Package Managers":  result.PackageManagers,
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

	// Save VS Code path if found
	if ScanResult.VSCodePath != "" {
		if err := config.SetConfigValue("vscode_path", ScanResult.VSCodePath); err != nil {
			return err
		}
	}

	return nil
}
