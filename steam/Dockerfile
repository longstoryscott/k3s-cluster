FROM ghcr.io/selkies-project/nvidia-glx-desktop:24.04
# Switch to root for package installation
USER root

# Set environment to non-interactive to avoid prompts during build
ENV DEBIAN_FRONTEND=noninteractive

# Update package lists and upgrade system
RUN apt-get update && \
  apt-get full-upgrade -y && \
  apt-get install -y \

  # 32-bit and 64-bit GL libraries for Steam
  libgl1-mesa-dri:i386 \
  libgl1:amd64 \
  libgl1:i386 \
  libgbm1:amd64 \
  libgbm1:i386 \
  steam-libs-amd64:amd64 \
  steam-libs-i386:i386 \
  xdg-desktop-portal \
  xdg-desktop-portal-kde \

  # Common utilities
  flatpak \
  wget \
  gpg \
  git \
  ca-certificates

# RUN flatpak remote-add --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo 
# RUN flatpak install flathub com.valvesoftware.Steam -y

RUN cd /tmp && \
  wget -O steam.deb "https://cdn.akamai.steamstatic.com/client/installer/steam.deb" && \
  wget -O vscode.deb "https://update.code.visualstudio.com/latest/linux-deb-x64/stable" && \
  apt-get install -y ./vscode.deb && \
  apt-get install -y ./steam.deb && \
  # Clean up to reduce image size
  update-alternatives --set editor /usr/bin/code && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*


# Replace steamdeps with a no-op script to skip interactive dependency checks
# Dependencies are already installed during image build
RUN mv /usr/bin/steamdeps /usr/bin/steamdeps.real && \
  echo '#!/bin/bash\n\
  # Dependencies already satisfied during image build\n\
  exit 0' > /usr/bin/steamdeps && \
  chmod +x /usr/bin/steamdeps

# Ensure ubuntu user owns their home directory
RUN chown -R ubuntu:ubuntu /home/ubuntu

# Switch back to ubuntu user
USER ubuntu

# Reset to default frontend
ENV DEBIAN_FRONTEND=dialog

# Metadata
LABEL maintainer="k3s-cluster"
LABEL description="Selkies GLX Desktop with Steam pre-installed - Software encoding"
LABEL version="1.0"
