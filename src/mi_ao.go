package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// getAlsaVolume reads current Soft Volume Master level (0-100).
// Output format: "Front Left: 64 [25%]" — we parse the bracketed percentage.
func getAlsaVolume() int {
	out, err := exec.Command("amixer", "sget", "Soft Volume Master").Output()
	if err != nil {
		return -1
	}
	s := string(out)
	idx := strings.Index(s, "Front Left:")
	if idx < 0 {
		return -1
	}
	// Skip the raw value and parse the percentage: "Front Left: 64 [25%]"
	var raw, pct int
	if n, _ := fmt.Sscanf(s[idx:], "Front Left: %d [%d%%]", &raw, &pct); n < 2 {
		return -1
	}
	return pct
}


// setAlsaVolume sets Soft Volume Master level (0-100)
func setAlsaVolume(vol int) {
        exec.Command("amixer", "sset", "Soft Volume Master", fmt.Sprintf("%d%%", vol)).Run()
}
