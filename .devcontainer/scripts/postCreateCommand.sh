#!/bin/bash
set -euo pipefail

# Installs and state that need to run per-container-creation go here.
# Tool installs are baked into the Dockerfile for caching.
sudo chown -R vscode:vscode /home/vscode/.local/share/opencode