#!/bin/sh
cd "$(dirname "$0")"
export LD_LIBRARY_PATH="./libs:/mnt/SDCARD/App/gmu/lib:/usr/miyoo/lib:/mnt/SDCARD/miyoo/lib:/usr/lib:/lib"
export SDL_VIDEODRIVER=a30
export SDL_NOMOUSE=1
killall -9 MainUI 2>/dev/null
sleep 1
# Set hardware DAC volume to max (resets to 0 when SDL_mixer closes audio device)
amixer sset "digital volume" 63 > /dev/null 2>&1
./MiyooPod >> ./miyoopod.log 2>&1
