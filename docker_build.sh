#!/bin/bash
set -e

docker build -t pikpakcli:latest .

echo "Docker image 'pikpakcli:latest' built successfully."