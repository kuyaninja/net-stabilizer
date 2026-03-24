// Package main provides OS-specific network recovery functions.
package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// FlushDNS executes the platform-specific command to clear the DNS resolver cache.
func FlushDNS() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("ipconfig", "/flushdns")
	case "darwin":
		// Flush standard cache
		err := exec.Command("dscacheutil", "-flushcache").Run()
		if err != nil {
			return err
		}
		// Restart mDNSResponder for macOS deep flush
		cmd = exec.Command("killall", "-HUP", "mDNSResponder")
	case "linux":
		cmd = exec.Command("resolvectl", "flush-caches")
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("flush dns failed: %s (output: %s)", err, string(output))
	}
	return nil
}

// SoftResetNetwork performs a platform-specific "soft" network reset.
// On Windows: Release/Renew IP.
// On macOS: Toggle Wi-Fi off and on.
// On Linux: Restart NetworkManager.
func SoftResetNetwork() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		err := exec.Command("ipconfig", "/release").Run()
		if err != nil {
			return err
		}
		cmd = exec.Command("ipconfig", "/renew")
	case "darwin":
		err := exec.Command("networksetup", "-setnetworkserviceenabled", "Wi-Fi", "off").Run()
		if err != nil {
			return fmt.Errorf("failed to turn off Wi-Fi: %v", err)
		}
		cmd = exec.Command("networksetup", "-setnetworkserviceenabled", "Wi-Fi", "on")
	case "linux":
		cmd = exec.Command("systemctl", "restart", "NetworkManager")
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("soft reset failed: %s (output: %s)", err, string(output))
	}
	return nil
}
