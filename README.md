# MiyooPod-A30

An automated port of [MiyooPod](https://github.com/danfragoso/miyoopod) (an iPod-inspired music player) for the **Miyoo A30** running **SpruceOS**.

This repo tracks the upstream MiyooPod project and automatically rebuilds for the A30 whenever a new version is released.

---

## Installation

1. Download the latest `MiyooPod-A30-vX.X.X.zip` from [Releases](../../releases)
2. Extract to `/mnt/SDCARD/App/` on your Miyoo A30 SD card
3. Launch MiyooPod from the PyUI menu

Your music library should be in `/mnt/SDCARD/Music/` (the same path MiyooPod uses on other devices).

---

## What Works

- ✅ Full display — all menus, bottom bar, progress bar, album art
- ✅ MP3 and OGG audio playback
- ✅ All buttons (A, B, X, Y, L, R, D-pad, Start, Select, Menu)
- ✅ Volume control via ALSA
- ✅ Brightness control
- ✅ Library scanning and album art
- ✅ Clean return to PyUI on exit

## Known Issues

- Sleep/wake: music seeks forward with no audio on resume
- Volume overlay indicator not visible

---

## How It Works

The A30 uses a portrait 480×640 framebuffer with a hardware rotation helper that presents it as 640×480 landscape to SDL apps via EGL. This causes SDL's `SDL_RenderCopy` to clip and distort output — only 75% of the screen is visible with the standard SDL rendering path.

The fix bypasses SDL rendering entirely and writes directly to `/dev/fb0` via mmap, applying a 90° CCW rotation and R/B channel swap (sunxi framebuffers use ARGB8888, MiyooPod renders ABGR8888). SDL is kept running only for input event handling and audio.

Other changes from upstream:
- **SDL2_mixer 2.6.3** built from source (upstream binary requires `Mix_MusicDuration` which isn't available in SpruceOS's bundled SDL2_mixer)
- **SDL2 with A30 driver** from GMU (system SDL2 lacks the `a30` video driver)
- **Volume control** via `amixer sset 'Soft Volume Master'` (replaces Miyoo Mini's proprietary MI_AO device)
- **Brightness control** via `/sys/devices/virtual/disp/disp/attr/lcdbl` (0–255 range)
- **Device detection** for 480×640 framebuffer → `miyoo-a30`

---

## Automated Builds

A GitHub Actions workflow runs daily and checks for new upstream releases. When a new version is found, it:
1. Clones the upstream repo at the new tag
2. Applies A30 patches
3. Cross-compiles for `arm-linux-gnueabihf` using a Debian Bullseye container with Go 1.22
4. Packages and publishes a new release here

You can also trigger a build manually from the [Actions tab](../../actions).

---

## Manual Build

If you want to build locally:

```sh
git clone https://github.com/amruthwo/MiyooPod-A30.git
cd MiyooPod-A30

# Clone upstream at desired tag
git clone --depth 1 --branch v0.0.6 https://github.com/danfragoso/miyoopod.git upstream
cd upstream

# Apply patches
patch -p1 < ../patches/logger.go.patch
patch -p1 < ../patches/ui.go.patch
patch -p1 < ../patches/power_mgmt.go.patch
cp ../overlay/src/main.c src/main.c
cp ../overlay/src/mi_ao.go src/mi_ao.go
cp ../overlay/App/MiyooPod/launch.sh App/MiyooPod/launch.sh
cp ../libs/libSDL2_mixer.so App/MiyooPod/libs/libSDL2_mixer.so
cp ../libs/libSDL2_mixer-2.0.so.0 App/MiyooPod/libs/libSDL2_mixer-2.0.so.0

# Build
docker build -f ../Dockerfile.a30 -t a30-cross ..
go run scripts/build-inject.go || true
docker run --rm -v "$(pwd)":/build \
  -e CC=arm-linux-gnueabihf-gcc \
  -e CGO_ENABLED=1 -e GOARCH=arm -e GOARM=7 -e GOOS=linux \
  a30-cross sh -c 'go build -a -o App/MiyooPod/MiyooPod src/*.go'
```

---

## Credits

- [danfragoso/miyoopod](https://github.com/danfragoso/miyoopod) — original MiyooPod by Dan Fragoso
- [SpruceOS](https://github.com/spruceUI/spruceOS) — firmware for Miyoo A30
- [ninoh-fox/SDL2_mali](https://github.com/ninoh-fox/sdl2_mali) — SDL2 with A30 driver (via GMU)
