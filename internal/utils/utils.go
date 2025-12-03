package utils

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Min returns the smaller of two ints
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CapitalizeFirst returns the string with the first letter capitalized
func CapitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// IsBinary reports whether data likely represents a binary file
func IsBinary(data []byte, maxCheckBytes int) bool {
	checkSize := Min(len(data), maxCheckBytes)
	for i := 0; i < checkSize; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// ReplacePlaceholders replaces all placeholders in content
func ReplacePlaceholders(content, projectName, author string, extraVars map[string]string) string {
	replacements := map[string]string{
		"{{PROJECT_NAME}}":       projectName,
		"{{AUTHOR}}":             author,
		"{{PROJECT_NAME_LOWER}}": strings.ToLower(projectName),
		"{{PROJECT_NAME_UPPER}}": strings.ToUpper(projectName),
	}
	for k, v := range extraVars {
		replacements["{{"+k+"}}"] = v
	}
	result := content
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// ParseVars parses --var key=value entries into a map
func ParseVars(kvs []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, kv := range kvs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid var format '" + kv + "', expected key=value")
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, errors.New("variable key cannot be empty")
		}
		result[key] = parts[1]
	}
	return result, nil
}

// LoadIgnorePatterns reads ignore patterns from a file
func LoadIgnorePatterns(root, filename string) []string {
	ignorePath := filepath.Join(root, filename)
	f, err := os.Open(ignorePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// MatchIgnore checks if a relative path matches any ignore pattern
func MatchIgnore(relPath string, patterns []string) bool {
	normalizedPath := filepath.ToSlash(relPath)
	for _, pattern := range patterns {
		normalizedPattern := filepath.ToSlash(strings.TrimSuffix(pattern, "/"))
		if matched, _ := filepath.Match(normalizedPattern, normalizedPath); matched {
			return true
		}
		if strings.HasPrefix(normalizedPath+"/", normalizedPattern+"/") {
			return true
		}
	}
	return false
}
