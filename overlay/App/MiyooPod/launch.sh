#!/bin/sh
cd "$(dirname "$0")"

export LD_LIBRARY_PATH="./libs:/mnt/SDCARD/App/gmu/lib:/usr/miyoo/lib:/mnt/SDCARD/miyoo/lib:/usr/lib:/lib"
export SDL_VIDEODRIVER=a30
export SDL_NOMOUSE=1

# Kill SpruceOS launcher cleanly
kill -9 $(pgrep -f principal.sh) 2>/dev/null
sleep 1
killall -9 MainUI 2>/dev/null
sleep 1

./MiyooPod >> ./miyoopod.log 2>&1

# Clear framebuffer and restart SpruceOS launcher
cat /dev/zero > /dev/fb0 2>/dev/null
cd /mnt/SDCARD
exec /mnt/SDCARD/spruce/scripts/principal.sh
