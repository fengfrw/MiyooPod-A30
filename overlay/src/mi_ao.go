package main

import (
	"fmt"
	"os/exec"
)

// setMiAOVolume sets system volume using amixer (A30/SpruceOS ALSA)
func setMiAOVolume(percent int) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Map 0-100% to 0-255 for 'Soft Volume Master'
	raw := percent * 255 / 100

	logMsg(fmt.Sprintf("DEBUG: Volume %d%% -> raw %d", percent, raw))

	cmd := exec.Command("amixer", "sset", "Soft Volume Master", fmt.Sprintf("%d", raw))
	if err := cmd.Run(); err != nil {
		logMsg(fmt.Sprintf("ERROR: amixer volume set failed: %v", err))
		return
	}

	logMsg(fmt.Sprintf("SUCCESS: amixer volume set: %d%% (raw %d)", percent, raw))
}
