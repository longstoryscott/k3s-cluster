# Docker Registry UI

This directory contains the configuration files needed to set up a web UI for your Docker Registry.

## Overview

The Docker Registry UI is a user interface for private docker registries. It provides:

- A simple, clean interface to browse your Docker Registry
- Ability to delete images (when configured)
- Display of image tags and metadata
- Search functionality
- Support for authentication

## Installation

To install the Docker Registry UI:

```bash
./install.sh
```

This script will:
1. Update the registry configuration to support CORS (needed for the UI)
2. Deploy the Docker Registry UI
3. Configure the UI to connect to your registry

## Accessing the UI

After installation, you can access the UI at:

```
http://<node-ip>:31580
```

Where `<node-ip>` is the IP address of any node in your K3s cluster.

## Authentication

The UI uses the same authentication as your Docker Registry. You'll need to provide the registry username and password when prompted.

## Features

- Browse images in your registry
- View image tags
- Delete images (if enabled)
- Search for images
- View image details and history

## Configuration

The UI is configured with the following settings:

- Single registry mode (no ability to switch between multiple registries)
- Image deletion enabled
- Content digest display enabled
- Catalog view with tag counts
- Automatic connection to your registry

## Troubleshooting

If you encounter issues:

1. Verify the registry pods are running:
   ```
   kubectl get pods -n registry
   ```

2. Check the registry-ui pod logs:
   ```
   kubectl logs -n registry deployment/registry-ui
   ```

3. Ensure the registry is accessible:
   ```
   curl -u <username>:<password> http://<node-ip>:31500/v2/_catalog
   ```

4. Verify CORS headers are correctly set:
   ```
   curl -I -X OPTIONS -u <username>:<password> http://<node-ip>:31500/v2/
   ```
