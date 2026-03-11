package main

import (
	"time"
)

// startPlaybackPoller checks audio state and updates progress display.
// Runs in its own goroutine. Minimal work per tick to avoid starving audio.
func (app *MiyooPod) startPlaybackPoller() {
	lastDrawnSecond := -1
	tickCount := 0
	saveTickCount := 0
	lastWallTime := time.Now()
	lastPosition := 0.0

	// Software clock for backends that don't report position (e.g., MP3/mpg123 on A30).
	// Tracks "position = clockBasePos at wall time clockBase".
	var clockBase time.Time    // Wall time at which position == clockBasePos
	var clockBasePos float64   // Position value anchored at clockBase
	var clockLastTrack *Track  // Track for which clock was last initialized
	var wasPaused bool         // Whether state was paused on the last tick
	var pauseStartWall time.Time // Wall time when pause began
	prevSeekActive := false    // Previous SeekActive value, to detect seek completion

	for app.Running {
		if app.Playing != nil && app.Playing.State != StateStopped {
			state := audioGetState()

			// Reset clock when a new track starts.
			if app.Playing.Track != clockLastTrack {
				clockLastTrack = app.Playing.Track
				clockBase = time.Now()
				clockBasePos = 0
				wasPaused = false
			}

			// Seek just completed: re-anchor clock to the position the seek landed on.
			if prevSeekActive && !app.SeekActive {
				clockBase = time.Now()
				clockBasePos = app.Playing.Position
				wasPaused = state.IsPaused
				if wasPaused {
					pauseStartWall = time.Now()
				}
			}
			prevSeekActive = app.SeekActive

			// Only trust backend position if it's strictly positive and actually advancing.
			// Mix_GetMusicPosition returns 0.0 (stuck) or -1.0 for MP3/mpg123 on A30.
			positionAdvancing := state.Position > 0 && state.Position != lastPosition
			if positionAdvancing && !app.SeekActive {
				app.Playing.Position = state.Position
				// Keep software clock anchored to real position when available.
				clockBase = time.Now()
				clockBasePos = state.Position
			} else if !app.SeekActive && !clockBase.IsZero() {
				// Backend doesn't report position (returns -1): use wall-clock estimate.
				if state.IsPlaying {
					if wasPaused {
						// Resumed: shift clock base forward by the paused duration.
						clockBase = clockBase.Add(time.Since(pauseStartWall))
						wasPaused = false
					}
					pos := clockBasePos + time.Since(clockBase).Seconds()
					if app.Playing.Duration > 0 && pos > app.Playing.Duration {
						pos = app.Playing.Duration
					}
					app.Playing.Position = pos
				} else if state.IsPaused && !wasPaused {
					wasPaused = true
					pauseStartWall = time.Now()
				}
			}

			// Detect post-sleep drift: position advancing faster than wall time.
			now := time.Now()
			wallElapsed := now.Sub(lastWallTime).Seconds()
			posElapsed := state.Position - lastPosition
			if wallElapsed > 0.5 && posElapsed > wallElapsed*1.5 && state.IsPlaying {
				logMsg("INFO: Audio drift detected (post-sleep), reinitializing audio")
				audioReinit()
			}
			lastWallTime = now
			lastPosition = state.Position
			if state.Duration > 0 && app.Playing.Track != nil && app.Playing.Track.Duration == 0 {
				app.Playing.Track.Duration = state.Duration
			}
			// Sync Playing.Duration from track (Mix_MusicDuration unreliable)
			if app.Playing.Duration == 0 && app.Playing.Track != nil && app.Playing.Track.Duration > 0 {
				app.Playing.Duration = app.Playing.Track.Duration
			}

			if state.IsPaused && app.Playing.State != StatePaused {
				app.Playing.State = StatePaused
				app.NPCacheDirty = true
				app.requestRedraw()
			} else if state.IsPlaying && app.Playing.State != StatePlaying {
				app.Playing.State = StatePlaying
				app.NPCacheDirty = true
				app.requestRedraw()
			}

			if state.Finished {
				app.handleTrackEnd()
			}

			// Update progress bar when on Now Playing screen and second changes
			if app.CurrentScreen == ScreenNowPlaying {
				currentSecond := int(app.Playing.Position)
				if currentSecond != lastDrawnSecond {
					lastDrawnSecond = currentSecond
					app.updateProgressBarOnly()
				}
			}

			// Redraw lyrics screen only when the highlighted LRC line changes,
			// and only when the user is not holding a scroll key (avoids flash during scroll).
			if app.CurrentScreen == ScreenLyrics && app.LyricsCachedLRC != nil && app.LastKey == NONE {
				activeLRC := activeLRCIndex(app.LyricsCachedLRC, app.Playing.Position)
				if activeLRC != app.LyricsLastActiveLRC {
					app.LyricsLastActiveLRC = activeLRC
					app.requestRedraw()
				}
			}

			// Flush audio buffers every 5 seconds to prevent choppy playback
			// Mimics the fix that happens when user manually pauses/resumes
			tickCount++
			if tickCount >= 5 {
				audioFlushBuffers()
				tickCount = 0
			}

			// Save playback state every 60 seconds during active playback
			saveTickCount++
			if saveTickCount >= 60 {
				app.savePlaybackState()
				saveTickCount = 0
			}
		}
		// Increased sleep to reduce CPU usage and SD card contention
		time.Sleep(1000 * time.Millisecond)
	}
}

func (app *MiyooPod) mpvLoadFile(path string) error {
	// Load file into RAM to eliminate SD card I/O during playback and seek
	err := audioLoadFileToMemory(path)
	if err != nil {
		// Fallback to streaming if memory load fails (e.g. very large file, low RAM)
		logMsg("WARN: memory load failed, falling back to streaming: " + err.Error())
		err = audioLoadFile(path)
		if err != nil {
			return err
		}
	}
	err = audioPlay()
	if err != nil {
		return err
	}
	// Mix_MusicDuration unreliable for some formats - use track duration from library scan
	if app.Playing != nil && app.Playing.Track != nil && app.Playing.Track.Duration > 0 {
		app.Playing.Duration = app.Playing.Track.Duration
	}
	return nil
}

func (app *MiyooPod) mpvTogglePause() {
	audioTogglePause()
}

func (app *MiyooPod) mpvStop() {
	audioStop()
}

func (app *MiyooPod) mpvSeekAbsolute(position float64) {
	if app.Playing == nil {
		return
	}
	if position < 0 {
		position = 0
	}
	if position > app.Playing.Duration && app.Playing.Duration > 0 {
		position = app.Playing.Duration
	}
	audioSeek(position)
}

func (app *MiyooPod) mpvSeek(seconds float64) {
	if app.Playing == nil {
		return
	}
	newPos := app.Playing.Position + seconds
	if newPos < 0 {
		newPos = 0
	}
	if newPos > app.Playing.Duration && app.Playing.Duration > 0 {
		newPos = app.Playing.Duration
	}
	audioSeek(newPos)
}
