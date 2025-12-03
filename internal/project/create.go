package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kajvans/foundry/internal/config"
	"github.com/kajvans/foundry/internal/utils"
)

// CreateFromTemplate copies the template to the target directory with placeholder replacement
func CreateFromTemplate(tmpl *config.Template, projectName, targetDir, author string, extraVars map[string]string) error {
	if err := ensureTargetDir(targetDir); err != nil {
		return err
	}

	absTargetDir, absSourceDir, err := resolvePaths(targetDir, tmpl.Path)
	if err != nil {
		return err
	}

	targetInsideSource := isTargetInsideSource(absSourceDir, absTargetDir)

	ignores := utils.LoadIgnorePatterns(absSourceDir, ".foundryignore")

	return copyTree(tmpl.Path, targetDir, absSourceDir, targetInsideSource, projectName, author, extraVars, ignores)
}

// PreviewSummary holds information about what would be generated
type PreviewSummary struct {
	ProjectName string
	TargetDir   string
	Template    string
	Language    string
	Files       []string
}

// PreviewFromTemplate walks the template and reports planned file outputs without writing
func PreviewFromTemplate(tmpl *config.Template, projectName, targetDir, author string, extraVars map[string]string) (*PreviewSummary, error) {
	absTargetDir, absSourceDir, err := resolvePaths(targetDir, tmpl.Path)
	if err != nil {
		return nil, err
	}
	targetInsideSource := isTargetInsideSource(absSourceDir, absTargetDir)
	ignores := utils.LoadIgnorePatterns(absSourceDir, ".foundryignore")

	files := []string{}
	err = filepath.Walk(tmpl.Path, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && shouldSkipDir(info.Name()) {
			return filepath.SkipDir
		}
		if targetInsideSource {
			relSrcFromSource, _ := filepath.Rel(absSourceDir, srcPath)
			relTarget, _ := filepath.Rel(absSourceDir, targetDir)
			isTargetOrChild := relSrcFromSource == relTarget || strings.HasPrefix(relSrcFromSource+string(os.PathSeparator), relTarget+string(os.PathSeparator))
			if isTargetOrChild {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		relPath, err := filepath.Rel(tmpl.Path, srcPath)
		if err != nil {
			return err
		}
		if utils.MatchIgnore(filepath.ToSlash(relPath), ignores) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if relPath == "." {
			return nil
		}
		dstPath := filepath.Join(targetDir, relPath)
		files = append(files, dstPath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &PreviewSummary{
		ProjectName: projectName,
		TargetDir:   targetDir,
		Template:    tmpl.Name,
		Language:    tmpl.Language,
		Files:       files,
	}, nil
}

func ensureTargetDir(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

func resolvePaths(targetDir, sourceDir string) (string, string, error) {
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to get absolute target path: %w", err)
	}
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to get absolute source path: %w", err)
	}
	if realTarget, err := filepath.EvalSymlinks(absTargetDir); err == nil {
		absTargetDir = realTarget
	}
	if realSource, err := filepath.EvalSymlinks(absSourceDir); err == nil {
		absSourceDir = realSource
	}
	return absTargetDir, absSourceDir, nil
}

func isTargetInsideSource(absSourceDir, absTargetDir string) bool {
	relTarget, relErr := filepath.Rel(absSourceDir, absTargetDir)
	return relErr == nil && !strings.HasPrefix(relTarget, "..")
}

func copyTree(sourceRoot, targetRoot, absSourceDir string, targetInsideSource bool, projectName, author string, extraVars map[string]string, ignores []string) error {
	walker := func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if skip, skipDir := shouldSkipEntry(info, srcPath, sourceRoot, targetRoot, absSourceDir, targetInsideSource, ignores); skip {
			if skipDir {
				return filepath.SkipDir
			}
			return nil
		}
		dstPath := joinDest(targetRoot, sourceRoot, srcPath)
		if info.IsDir() {
			return ensureDir(dstPath, info.Mode())
		}
		return copyFileWithReplacements(srcPath, dstPath, projectName, author, info.Mode(), extraVars)
	}
	return filepath.Walk(sourceRoot, walker)
}

func shouldSkipEntry(info os.FileInfo, srcPath, sourceRoot, targetRoot, absSourceDir string, targetInsideSource bool, ignores []string) (skip bool, skipDir bool) {
	if info.IsDir() && shouldSkipDir(info.Name()) {
		return true, true
	}
	if targetInsideSource && isTargetOrChild(srcPath, absSourceDir, targetRoot) {
		if info.IsDir() {
			return true, true
		}
		return true, false
	}
	relPath, err := filepath.Rel(sourceRoot, srcPath)
	if err != nil {
		return false, false
	}
	if relPath == "." {
		return true, false
	}
	if utils.MatchIgnore(filepath.ToSlash(relPath), ignores) {
		if info.IsDir() {
			return true, true
		}
		return true, false
	}
	return false, false
}

func isTargetOrChild(srcPath, absSourceDir, targetRoot string) bool {
	relSrcFromSource, _ := filepath.Rel(absSourceDir, srcPath)
	relTarget, _ := filepath.Rel(absSourceDir, targetRoot)
	return relSrcFromSource == relTarget || strings.HasPrefix(relSrcFromSource+string(os.PathSeparator), relTarget+string(os.PathSeparator))
}

func joinDest(targetRoot, sourceRoot, srcPath string) string {
	relPath, _ := filepath.Rel(sourceRoot, srcPath)
	return filepath.Join(targetRoot, relPath)
}

func ensureDir(path string, mode os.FileMode) error {
	return os.MkdirAll(path, mode)
}

func shouldSkipDir(name string) bool {
	switch name {
	case "node_modules", "vendor", ".venv", "dist", "build", ".git":
		return true
	}
	return false
}

func copyFileWithReplacements(src, dst, projectName, author string, mode os.FileMode, extraVars map[string]string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}
	if utils.IsBinary(content, 8000) { // use same default as cmd
		return os.WriteFile(dst, content, mode)
	}
	contentStr := utils.ReplacePlaceholders(string(content), projectName, author, extraVars)
	return os.WriteFile(dst, []byte(contentStr), mode)
}
