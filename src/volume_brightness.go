package main

import (
	"fmt"
	"time"
)

// adjustVolume changes system volume and shows overlay
func (app *MiyooPod) adjustVolume(delta int) {
	newVolume := clamp(app.SystemVolume+delta, 0, 100)
	setAlsaVolume(newVolume)
	// Read back what ALSA actually set in case SpruceOS's daemon also incremented
	// on the same event, causing our tracked value to drift from reality.
	if actual := getAlsaVolume(); actual >= 0 {
		app.SystemVolume = actual
	} else {
		app.SystemVolume = newVolume
	}
	app.showOverlay("volume", app.SystemVolume)
	logMsg(fmt.Sprintf("Volume: %d%%", app.SystemVolume))
}
func (app *MiyooPod) adjustBrightness(delta int) {
	newBrightness := clamp(app.SystemBrightness+delta, 10, 100) // Min 10% so screen stays visible

	setBrightness(newBrightness)
	app.SystemBrightness = newBrightness

	// Re-sync volume: SpruceOS may also handle the SELECT+VOLUME combo as a plain
	// volume event, causing ALSA to change as a side effect of brightness presses.
	if actual := getAlsaVolume(); actual >= 0 {
		app.SystemVolume = actual
	}

	app.showOverlay("brightness", newBrightness)

	// Persist to settings
	go app.saveSettings()

	logMsg(fmt.Sprintf("Brightness: %d%%", newBrightness))
}

// showOverlay displays the volume/brightness overlay for 2 seconds
func (app *MiyooPod) showOverlay(overlayType string, value int) {
	// Cancel existing timer
	if app.OverlayTimer != nil {
		app.OverlayTimer.Stop()
	}

	app.OverlayType = overlayType
	app.OverlayValue = value
	app.OverlayVisible = true

	// Signal main loop to redraw (non-blocking to avoid deadlock)
	app.requestRedraw()

	// Hide after 2 seconds
	app.OverlayTimer = time.AfterFunc(2*time.Second, func() {
		app.OverlayVisible = false
		app.requestRedraw()
	})
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
