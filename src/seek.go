package main

import (
	"time"
)

const (
	seekHoldThreshold = 400 * time.Millisecond
	seekTickInterval  = 100 * time.Millisecond
)

func (app *MiyooPod) seekAmount() float64 {
	held := time.Since(app.SeekStartTime).Seconds()
	switch {
	case held < 5:
		return 5.0
	case held < 10:
		return 10.0
	default:
		return 30.0
	}
}

func (app *MiyooPod) seekKeyPressed(direction int) {
	if app.SeekHeld {
		return
	}
	app.SeekHeld = true
	app.SeekActive = false
	app.SeekDirection = direction
	app.SeekStartTime = time.Now()
	app.LastSeekTick = time.Time{}
	app.SeekPreviewPos = app.Playing.Position
}

func (app *MiyooPod) seekKeyReleased() int {
	if !app.SeekHeld {
		return 0
	}
	direction := app.SeekDirection
	wasActive := app.SeekActive
	previewPos := app.SeekPreviewPos
	app.SeekHeld = false
	app.SeekDirection = 0
	app.SeekStartTime = time.Time{}
	app.LastSeekTick = time.Time{}
	if wasActive {
		app.RestoreSeekTarget = 0 // manual seek overrides any deferred restore seek
		app.Playing.Position = previewPos
		app.SeekLoading = true
		app.drawSeekToast()
		app.triggerRefresh()
		app.mpvSeekAbsolute(previewPos)
		// Set SeekActive false AFTER seek completes so the wall-clock re-anchor in
		// startPlaybackPoller fires at the correct time (when audio actually starts),
		// not partway through the seek scan where it races ahead of the audio.
		app.SeekActive = false
		app.SeekLoading = false
		app.requestRedraw()
		return 0
	}
	app.SeekActive = false
	return direction
}

func (app *MiyooPod) pollSeek() {
	if !app.SeekHeld || app.SeekStartTime.IsZero() {
		return
	}
	if app.CurrentScreen != ScreenNowPlaying || app.Playing == nil ||
		(app.Playing.State != StatePlaying && app.Playing.State != StatePaused) {
		app.SeekHeld = false
		app.SeekActive = false
		return
	}
	elapsed := time.Since(app.SeekStartTime)
	if elapsed < seekHoldThreshold {
		return
	}
	if !app.SeekActive {
		app.SeekActive = true
		app.LastSeekTick = time.Now()
		app.performSeekTick()
		return
	}
	if time.Since(app.LastSeekTick) >= seekTickInterval {
		app.LastSeekTick = time.Now()
		app.performSeekTick()
	}
}

func (app *MiyooPod) performSeekTick() {
	if app.Playing == nil || app.Playing.State == StateStopped {
		return
	}
	amount := app.seekAmount() * float64(app.SeekDirection)
	newPos := app.SeekPreviewPos + amount
	if newPos < 0 {
		newPos = 0
	}
	if app.Playing.Duration > 0 && newPos > app.Playing.Duration {
		newPos = app.Playing.Duration
	}
	app.SeekPreviewPos = newPos
	app.Playing.Position = newPos
	if app.CurrentScreen == ScreenNowPlaying {
		app.updateProgressBarOnly()
	}
}
