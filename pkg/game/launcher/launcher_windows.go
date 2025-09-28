//go:build windows

package launcher

import (
	"os/exec"
	"syscall"
)

// setupWindowsProcessAttributes sets up Windows-specific process attributes
// to hide the console window when launching Minecraft
func setupWindowsProcessAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
