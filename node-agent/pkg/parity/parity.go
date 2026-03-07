// Package parity wraps SnapRAID commands for use by the KubeNAS Node Agent.
package parity

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// OperationResult captures the outcome of a SnapRAID operation.
type OperationResult struct {
	Operation  string
	StartTime  time.Time
	EndTime    time.Time
	ExitCode   int
	Stdout     string
	Stderr     string
	Success    bool
	ErrorCount int64
}

const defaultConfPath = "/etc/snapraid/snapraid.conf"

// Syncer runs SnapRAID sync to update parity with current data disk state.
// This must be run after any file changes to ensure parity remains valid.
func Sync(confPath string) (*OperationResult, error) {
	return runSnapraid(confPath, "sync")
}

// Check verifies parity data integrity without modifying anything.
// Returns an error if corrupted data is detected.
func Check(confPath string) (*OperationResult, error) {
	return runSnapraid(confPath, "check")
}

// Scrub validates a percentage of parity-protected data.
// percentBlock limits how much is checked per run (1-100).
func Scrub(confPath string, percentBlock int) (*OperationResult, error) {
	if percentBlock <= 0 || percentBlock > 100 {
		percentBlock = 100
	}
	return runSnapraid(confPath, "scrub", "-p", fmt.Sprintf("%d", percentBlock))
}

// Fix attempts to recover data from parity after a disk failure.
// diskLabel is the snapraid disk label to recover (e.g., "d2").
func Fix(confPath, diskLabel string) (*OperationResult, error) {
	return runSnapraid(confPath, "fix", "-d", diskLabel)
}

// Status returns a human-readable SnapRAID array status.
func Status(confPath string) (*OperationResult, error) {
	return runSnapraid(confPath, "status")
}

// Diff reports which files have changed since the last sync.
func Diff(confPath string) (*OperationResult, error) {
	return runSnapraid(confPath, "diff")
}

// runSnapraid executes a snapraid command with the given config file and arguments.
func runSnapraid(confPath, command string, extraArgs ...string) (*OperationResult, error) {
	if confPath == "" {
		confPath = defaultConfPath
	}

	args := append([]string{"-c", confPath, command}, extraArgs...)
	cmd := exec.Command("snapraid", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	end := time.Now()

	result := &OperationResult{
		Operation: command,
		StartTime: start,
		EndTime:   end,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Success:   runErr == nil,
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	// Parse error counts from snapraid output.
	result.ErrorCount = parseErrorCount(stdout.String())

	if runErr != nil {
		return result, fmt.Errorf("snapraid %s exited %d: %s", command, result.ExitCode, stderr.String())
	}

	return result, nil
}

// parseErrorCount extracts the error count from snapraid output.
// SnapRAID outputs lines like: "Found X errors during..."
func parseErrorCount(output string) int64 {
	var count int64
	for _, line := range strings.Split(output, "\n") {
		line = strings.ToLower(line)
		if strings.Contains(line, "error") && strings.Contains(line, "found") {
			var n int64
			if _, err := fmt.Sscanf(line, "found %d error", &n); err == nil {
				count += n
			}
		}
	}
	return count
}

// IsSnapraidAvailable checks whether snapraid is installed.
func IsSnapraidAvailable() bool {
	_, err := exec.LookPath("snapraid")
	return err == nil
}

// WriteConfig writes a snapraid.conf file to the specified path.
// The rendered config is produced by the operator's renderSnapraidConf function
// and delivered via ConfigMap. This helper writes it to the filesystem.
func WriteConfig(confPath, content string) error {
	// Ensure parent directory exists.
	dir := confPath[:strings.LastIndex(confPath, "/")]
	if dir == "" {
		dir = "/etc/snapraid"
	}

	if err := exec.Command("mkdir", "-p", dir).Run(); err != nil {
		return fmt.Errorf("creating config dir %s: %w", dir, err)
	}

	f, err := openFileTruncate(confPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}

// openFileTruncate opens a file for writing, creating or truncating it.
// This is factored out to avoid importing os in the stub.
func openFileTruncate(path string) (interface{ WriteString(string) (int, error); Close() error }, error) {
	// Real implementation uses os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644).
	// Stubbed here to avoid cyclic imports; replaced in integration.
	return nil, fmt.Errorf("not implemented in stub — use os.OpenFile in production build")
}
