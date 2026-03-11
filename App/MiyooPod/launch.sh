#!/bin/sh
cd "$(dirname "$0")"
export LD_LIBRARY_PATH="./libs:/mnt/SDCARD/App/gmu/lib:/usr/miyoo/lib:/mnt/SDCARD/miyoo/lib:/usr/lib:/lib"
export SDL_VIDEODRIVER=a30
export SDL_NOMOUSE=1
# Swap in any staged .new libs from OTA update
for f in ./libs/*.new; do
    [ -f "$f" ] && mv "$f" "${f%.new}"
done
./MiyooPod >> ./miyoopod.log 2>&1
sync
