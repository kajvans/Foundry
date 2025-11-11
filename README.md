````markdown
# Foundry

A fast, flexible CLI to manage project templates and scaffold new projects with smart detection, interactive selection, and safe copy semantics.

- Language/framework tagging on templates (e.g., Go, React, Vue)
- Arrow-key interactive menus for choosing language and template
- Per-language default templates and non-interactive automation
- JSON outputs and quiet modes for scripting
- `.foundryignore` support, symlink-safe copying, and binary-safe replacements
- Git repository initialization with language-specific `.gitignore` files
- Clone templates directly from Git repositories

## Installation

### Windows

1. Run `foundry-0.1.0-setup.exe` from the releases.  
2. Follow the installer prompts.  
3. The executable will be installed and optionally added to your PATH.  
4. Verify installation:

```powershell
foundry --version
foundry --help
````

### Linux

#### Using the tarball

1. Extract the tarball:

```bash
tar -xzf foundry-0.1.0-linux-amd64.tar.gz
# or for ARM64
tar -xzf foundry-0.1.0-linux-arm64.tar.gz
```

2. Run the included install script:

```bash
cd foundry-0.1.0
sudo ./install.sh
```

#### Using the Debian package

```bash
sudo dpkg -i foundry-0.1.0-amd64.deb
# or for ARM64
sudo dpkg -i foundry-0.1.0-arm64.deb
```

3. The executable will be installed system-wide (or as defined in the script).

Verify installation:

```bash
foundry --version
foundry --help
```

## Quick start

1. Add a template from a folder (auto-detects language):

```powershell
foundry template add foundry-cli ./examples/foundry-cli --description "Go CLI starter" --language Go
```

2. Tag frameworks like React or Vue:

```powershell
foundry template add react-starter C:\templates\react-app --description "React + TS" --language React
```

3. Set a default template for a language (optional):

```powershell
foundry config Go foundry-cli
```

4. Create a new project interactively:

```powershell
foundry new my-app
```

5. Or non-interactively:

```powershell
# Use default Go template
foundry new my-api --language Go

# Use a specific template
foundry new my-react --template react-starter
```

## Commands

### Global flags

* `--config <path>`: Use a custom config path (default: `~/.foundry/config.yaml`)
* `--no-color`: Disable colored output
* `--color`: Force colored output (overrides `NO_COLOR` environment variable)
* `--version` / `-v`: Print version and exit

**Color control:**

* By default, colored output is enabled
* Set `NO_COLOR` environment variable to disable colors globally
* Use `--no-color` flag to disable colors for a single command
* Use `--color` flag to force colors even when `NO_COLOR` is set
* Flag takes precedence over environment variable

### detect

Detect languages, package managers, and dev tools on your system.

```powershell
foundry detect
foundry detect --json
foundry detect --non-interactive --yes
```

* `--json`: machine-readable output
* `--non-interactive`: do not prompt
* `--yes`: auto-save results when non-interactive

### template

Manage project templates.

* **Add**:

```powershell
foundry template add <name> <path> [--description <text>] [--language <tag>]
```

* **List**:

```powershell
foundry template list [--sort name|language] [--quiet]
```

* **Show**:

```powershell
foundry template show <name> [--files-only] [--summary] [--json]
```

* **Remove**:

```powershell
foundry template remove <name> [--force]
```

### new

Create a new project from a saved template or clone from a Git repository:

```powershell
foundry new <project-name> \
  [--language <Lang>] [--template <Name>] [--git <URL>] \
  [--path <Dir>] [--no-git] [--non-interactive] \
  [--var KEY=VALUE ...]
```

**Behavior**:

* `--language`: uses the default template for that language
* `--template`: uses a specific template
* `--git`: clones a template from a Git repository URL
* Interactive mode shows two menus if none of the above is provided

**Examples**:

```powershell
# Use default Go template
foundry new my-api --language Go

# Use a specific saved template
foundry new my-app --template react-starter

# Clone template from GitHub
foundry new my-project --git https://github.com/username/template-repo

# Create in custom location without git init
foundry new my-app --language Python --path ~/projects --no-git
```

**Flags**:

* `--path`: target directory (default: `./<project-name>`)
* `--no-git`: skip git initialization
* `--non-interactive`: disable menus
* `--var KEY=VALUE`: replace custom placeholders in text files

**Git features**:

* Automatically initializes git repository in new projects
* Downloads language-specific `.gitignore` from [github/gitignore](https://github.com/github/gitignore)
* Creates initial commit with "Initial commit from Foundry"
* Use `--no-git` flag to skip git initialization

**Placeholders replaced**:

* `{{PROJECT_NAME}}`, `{{AUTHOR}}`, `{{PROJECT_NAME_LOWER}}`, `{{PROJECT_NAME_UPPER}}`, plus any custom `--var KEY=VALUE`

**Safeguards**:

* Symlink/junction-safe copying
* Skips heavy directories (`node_modules`, `vendor`, `.venv`, `dist`, `build`)
* Respects `.foundryignore`
* Binary-safe replacements

## .foundryignore

Place at the root of a template to exclude files/folders from scanning and copying. Simple glob/prefix matching.

```gitignore
# Ignore build outputs
/dist/
/build/

# Ignore large directories
node_modules/
vendor/

# Ignore dotfiles
.*
```

## Configuration

* Default config file: `~/.foundry/config.yaml`
* Stores saved templates and language defaults

Commands:

```powershell
# Set default template
foundry config Go foundry-cli

# Clear a language default
foundry config Go ""
```

## Tips

* Disable color output via `--no-color` or `NO_COLOR` environment variable
* Windows: use terminals that support ANSI sequences (Windows Terminal, VS Code, PowerShell 7+)

## Roadmap

* Richer ignore pattern semantics
* Template edit command (update language tag, description, path)
* Git branch/tag selection for `--git` flag

## License

See [LICENSE](LICENSE)
