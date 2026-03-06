#!/bin/bash
set -e

# Restrict existing sources to amd64 only (armhf not available on azure/security mirrors)
sed -i 's/^deb http/deb [arch=amd64] http/g' /etc/apt/sources.list
sed -i 's/^deb https/deb [arch=amd64] https/g' /etc/apt/sources.list

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
