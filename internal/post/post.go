package post

import (
	"os/exec"
)

// RunLanguagePost executes language-specific setup commands inside projectDir.
// It is safe: failures do not abort; they return error to be handled by caller.
func RunLanguagePost(language, projectDir string) error {
	var cmd *exec.Cmd
	switch language {
	case "Go":
		cmd = exec.Command("bash", "-lc", "cd \""+projectDir+"\" && go mod tidy && go build")
	case "JavaScript", "TypeScript", "React":
		cmd = exec.Command("bash", "-lc", "cd \""+projectDir+"\" && npm install && npm run dev")
	case "Python":
		cmd = exec.Command("bash", "-lc", "cd \""+projectDir+"\" && (test -f requirements.txt && pip install -r requirements.txt || true) && python main.py")
	default:
		return nil
	}
	return cmd.Run()
}
