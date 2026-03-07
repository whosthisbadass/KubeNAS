package snapraid

import (
	"fmt"
	"os"
	"os/exec"
)

func WriteConfig(path string, cfg Config) error {
	return os.WriteFile(path, []byte(Render(cfg)), 0o644)
}

func Run(cmdName string) error {
	allowed := map[string]bool{"sync": true, "scrub": true, "fix": true}
	if !allowed[cmdName] {
		return fmt.Errorf("unsupported snapraid command: %s", cmdName)
	}
	cmd := exec.Command("snapraid", cmdName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("snapraid %s failed: %w (%s)", cmdName, err, string(out))
	}
	return nil
}
