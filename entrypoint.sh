#!/bin/sh
#Path to the mounted volume where the config will be placed
CONFIG_PATH="/app/config/config.json"

# Check if the config file in the volume exists
if [ ! -f "$CONFIG_PATH" ]; then
    echo "Config file not found in the volume, copying the default config..."
    # Copy the default config file to the volume
    cp /app/config_template.json "$CONFIG_PATH"
else
    echo "Config file already exists, skipping copy."
fi

# Run the main application
exec /app/plunger-server
