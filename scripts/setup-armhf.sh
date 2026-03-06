#!/bin/bash
set -e

# Restrict ALL existing apt sources to amd64 only
find /etc/apt/sources.list /etc/apt/sources.list.d/ -name "*.list" | \
    xargs sed -i 's/^deb \[/deb [arch=amd64 /g; s/^deb http/deb [arch=amd64] http/g; s/^deb https/deb [arch=amd64] https/g'

# Add armhf architecture
dpkg --add-architecture armhf

# Add ports.ubuntu.com for armhf only
cat > /etc/apt/sources.list.d/armhf-ports.list << 'EOF'
deb [arch=armhf] http://ports.ubuntu.com/ubuntu-ports jammy main universe
deb [arch=armhf] http://ports.ubuntu.com/ubuntu-ports jammy-updates main universe
deb [arch=armhf] http://ports.ubuntu.com/ubuntu-ports jammy-security main universe
EOF

apt-get update

apt-get install -y \
    gcc-arm-linux-gnueabihf \
    libsdl2-dev:armhf \
    libsdl2-mixer-dev:armhf

echo "Setup complete"
