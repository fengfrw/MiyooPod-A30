#!/bin/bash
set -e

# Add armhf architecture
dpkg --add-architecture armhf

# Add ports repo for armhf packages
cat > /etc/apt/sources.list.d/armhf-ports.list << 'EOF'
deb [arch=armhf] http://ports.ubuntu.com/ubuntu-ports jammy main universe
deb [arch=armhf] http://ports.ubuntu.com/ubuntu-ports jammy-updates main universe
EOF

apt-get update

# Install cross-compiler (amd64) and SDL2 + SDL2_mixer (armhf)
apt-get install -y \
    gcc-arm-linux-gnueabihf \
    libsdl2-dev:armhf \
    libsdl2-mixer-dev:armhf

echo "Setup complete"
ls /usr/lib/arm-linux-gnueabihf/libSDL2* 2>/dev/null || echo "WARNING: libSDL2 not found"
ls /usr/include/SDL2/SDL_mixer.h 2>/dev/null || \
ls /usr/include/arm-linux-gnueabihf/SDL2/SDL_mixer.h 2>/dev/null || \
echo "WARNING: SDL_mixer.h not found"
