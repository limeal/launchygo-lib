//go:build !windows

package launcher

import (
	"os/exec"
)

// setupWindowsProcessAttributes is a no-op on non-Windows systems
func setupWindowsProcessAttributes(cmd *exec.Cmd) {
	// No special setup needed on Unix-like systems
}
