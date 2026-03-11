package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

// Linux input event structure
type inputEvent struct {
	Time  syscallTimeval
	Type  uint16
	Code  uint16
	Value int32
}

type syscallTimeval struct {
	Sec  int32
	Usec int32
}

const (
	EV_KEY         = 0x01
	EV_ABS         = 0x03
	KEY_POWER      = 116
	KEY_VOLUMEUP   = 115
	KEY_VOLUMEDOWN = 114
	KEY_SELECT     = 97 // RIGHT_CTRL on gpio-keys-polled

	// Joystick axis thresholds (center X≈-20, center Y≈-16)
	JOY_X_LEFT  = -80
	JOY_X_RIGHT = 40
	JOY_Y_UP    = -50
	JOY_Y_DOWN  = 20
)

// startPowerButtonMonitor reads power button events from /dev/input/event0,
// volume/SELECT events from /dev/input/event3, and joystick events from /dev/input/event4.
func (app *MiyooPod) startPowerButtonMonitor() {
	// event0: power button (AXP power chip)
	go func() {
		file, err := os.Open("/dev/input/event0")
		if err != nil {
			logMsg(fmt.Sprintf("WARNING: Could not open /dev/input/event0: %v", err))
			return
		}
		defer file.Close()
		logMsg("Power button monitor started on /dev/input/event0")

		var ev inputEvent
		for app.Running {
			if err := binary.Read(file, binary.LittleEndian, &ev); err != nil {
				logMsg(fmt.Sprintf("ERROR: Reading event0: %v", err))
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if ev.Type != EV_KEY {
				continue
			}
			if ev.Code == KEY_POWER {
				if ev.Value == 1 {
					app.handlePowerButtonPress()
				} else if ev.Value == 0 {
					app.handlePowerButtonRelease()
				}
			}
		}
	}()

	// event3: gpio-keys-polled (volume up/down, SELECT for brightness combo)
	go func() {
		file, err := os.Open("/dev/input/event3")
		if err != nil {
			logMsg(fmt.Sprintf("WARNING: Could not open /dev/input/event3: %v", err))
			return
		}
		defer file.Close()
		logMsg("Button monitor started on /dev/input/event3")

		selectHeld := false
		var ev inputEvent
		for app.Running {
			if err := binary.Read(file, binary.LittleEndian, &ev); err != nil {
				logMsg(fmt.Sprintf("ERROR: Reading event3: %v", err))
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if ev.Type != EV_KEY {
				continue
			}
			switch ev.Code {
			case KEY_SELECT:
				selectHeld = ev.Value == 1
			case KEY_VOLUMEUP:
				if ev.Value == 1 && !app.Locked {
					if selectHeld {
						app.adjustBrightness(10)
					} else {
						app.adjustVolume(5)
					}
				}
			case KEY_VOLUMEDOWN:
				if ev.Value == 1 && !app.Locked {
					if selectHeld {
						app.adjustBrightness(-10)
					} else {
						app.adjustVolume(-5)
					}
				}
			}
		}
	}()

	// event4: MIYOO Pad1 joystick (analog stick axes → d-pad navigation)
	go func() {
		file, err := os.Open("/dev/input/event4")
		if err != nil {
			logMsg(fmt.Sprintf("WARNING: Could not open /dev/input/event4: %v", err))
			return
		}
		defer file.Close()
		logMsg("Joystick monitor started on /dev/input/event4")

		axisX, axisY := 0, 0
		dirX, dirY := 0, 0 // -1/0/1 for each axis

		var ev inputEvent
		for app.Running {
			if err := binary.Read(file, binary.LittleEndian, &ev); err != nil {
				logMsg(fmt.Sprintf("ERROR: Reading event4: %v", err))
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if ev.Type != EV_ABS {
				continue
			}

			if ev.Code == 0 {
				axisX = int(ev.Value)
			} else if ev.Code == 1 {
				axisY = int(ev.Value)
			} else {
				continue
			}

			// Determine new direction for each axis
			newDirX := 0
			if axisX < JOY_X_LEFT {
				newDirX = -1
			} else if axisX > JOY_X_RIGHT {
				newDirX = 1
			}

			newDirY := 0
			if axisY < JOY_Y_UP {
				newDirY = -1
			} else if axisY > JOY_Y_DOWN {
				newDirY = 1
			}

			// On direction change: send key and arm the existing d-pad repeat mechanism
			if newDirX != dirX {
				dirX = newDirX
				if dirX == -1 {
					app.sendJoystickKey(LEFT)
					app.LastKey = LEFT
					app.LastKeyTime = time.Now()
				} else if dirX == 1 {
					app.sendJoystickKey(RIGHT)
					app.LastKey = RIGHT
					app.LastKeyTime = time.Now()
				} else if app.LastKey == LEFT || app.LastKey == RIGHT {
					app.LastKey = NONE
				}
			}
			if newDirY != dirY {
				dirY = newDirY
				if dirY == -1 {
					app.sendJoystickKey(UP)
					app.LastKey = UP
					app.LastKeyTime = time.Now()
				} else if dirY == 1 {
					app.sendJoystickKey(DOWN)
					app.LastKey = DOWN
					app.LastKeyTime = time.Now()
				} else if app.LastKey == UP || app.LastKey == DOWN {
					app.LastKey = NONE
				}
			}
		}
	}()
}

// sendJoystickKey injects a directional key into the main event loop via JoystickChan.
func (app *MiyooPod) sendJoystickKey(key Key) {
	select {
	case app.JoystickChan <- key:
	default:
	}
}

func (app *MiyooPod) handlePowerButtonPress() {
	if !app.PowerButtonPressed {
		app.PowerButtonPressed = true
		app.PowerButtonPressTime = time.Now()
		// Start monitoring for long hold
		go app.monitorPowerButtonHold()
	}
}

func (app *MiyooPod) handlePowerButtonRelease() {
	if app.PowerButtonPressed {
		holdDuration := time.Since(app.PowerButtonPressTime)
		app.PowerButtonPressed = false

		// If held for less than 5 seconds, toggle lock
		if holdDuration < 5*time.Second {
			app.toggleLock()
		}
		// If held for 5+ seconds, monitorPowerButtonHold already handled shutdown
	}
}

