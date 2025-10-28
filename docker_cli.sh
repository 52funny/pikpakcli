#!/bin/bash

# Default directories for download and upload
DOWNLOAD_DIR="${PWD}/pikpak_downloads"
UPLOAD_DIR="${PWD}/pikpak_uploads"

CONFIG_PATH="${PWD}/config.yml"

# Create directories if they don't exist
mkdir -p "${DOWNLOAD_DIR}"
mkdir -p "${UPLOAD_DIR}"


if [ ! -f "${CONFIG_PATH}" ]; then
    echo "config.yml not found in ${CONFIG_PATH}. Please ensure it exists."
    echo "Exiting."
    exit 1
fi

# Run the container with mounted volumes and pass all arguments
docker run --rm -it \
    -v "${DOWNLOAD_DIR}":/pikpak_downloads \
    -v "${UPLOAD_DIR}":/pikpak_uploads \
    -v "${CONFIG_PATH}":/root/.config/pikpakcli/config.yml \
    pikpakcli:latest "$@"
