# LaunchyGo Lib

A powerful Go library for launching Minecraft with support for multiple mod loaders, authentication systems, and remote game folder management.

## Features

- 🎮 **Minecraft Launcher**: Launch Minecraft with custom configurations
- 🔧 **Mod Loader Support**: Generate and launch Vanilla and Fabric modpacks
- 🔐 **Authentication**: Support for Microsoft and custom authentication systems
- 🌐 **Remote Game Folders**: Download and manage game folders from SFTP, HTTP, and local file systems
- 📦 **Game Pack Management**: Generate, publish, and manage Minecraft game packs
- ⚡ **Multi-threaded Downloads**: Fast parallel downloading of game assets
- 🖥️ **Cross-platform**: Works on Windows, macOS, and Linux
- 🎯 **Quick Play**: Direct server joining with Quick Play support

## Installation

```bash
go get limeal.fr/launchygo
```

## Quick Start

### 1. Generate a Game Pack

First this librairies use a 3 layers flow:

1- Generate (download required files from mojang servers and put it in packs)
2 - Publish (Before publishing add extra file aka: `mods`, ... and then use it)
    - It will update the manifest and upload it to your file storage service through sftp or current folder
3 - Launch (Download from your server and tweak the game with your settings)

## Command Line Interface

### Global Options

- `--debug, -d`: Enable debug mode for verbose output

### Launch Command

```bash
launchygo launch <game_folder> <uri> [flags]
```

**Arguments:**
- `game_folder`: The name of the game folder to launch
- `uri`: The URI to the game folder (file://, sftp://, http://, https://)

**Flags:**
- `--Xmx, -x int`: Maximum memory allocation in GB (default: 4)
- `--Xms, -s int`: Initial memory allocation in GB (default: 2)
- `--java, -j string`: Path to Java executable
- `--auth, -a string`: Authentication URI (microsoft://code, custom://user:pass@base_url/login_endpoint)
- `--quickPlayMultiplayer string`: Server address for Quick Play

### Generate Command

```bash
launchygo generate <type> <pack_name> <version>
```

**Arguments:**
- `type`: Generator type (vanilla, fabric)
- `pack_name`: Name for the generated pack
- `version`: Minecraft version (e.g., 1.20.1)

### Publish Command

```bash
launchygo publish <pack_name> <uri>
```

**Arguments:**
- `pack_name`: Name of the pack to publish
- `uri`: Destination URI for publishing

## Library Usage

### Basic Launcher Setup

```go
package main

import (
    "fmt"
    "limeal.fr/launchygo/connectors"
    "limeal.fr/launchygo/game/folder"
    "limeal.fr/launchygo/game/launcher"
    "limeal.fr/launchygo/game/profile"
)

func main() {
    // Create a connector (file, sftp, http)
    connector := connectors.FindConnectorFromURI("file:///path/to/pack")
    if err := connector.Connect(); err != nil {
        panic(err)
    }
    defer connector.Close()

    // Initialize game folder
    gameFolder, err := folder.InitGameFolder(connector, "my-pack")
    if err != nil {
        panic(err)
    }

    // Create game profile
    profile := profile.NewGameProfile()
    profile.SetMemory(8, 4) // 8GB max, 4GB initial

    // Create launcher
    launcherInstance := launcher.NewLauncher()
    launcherInstance.SetGameFolder(gameFolder)
    launcherInstance.SetProfile(profile)

    // Build the game folder (downloads missing files)
    if err := gameFolder.Build(false, nil); err != nil {
        panic(err)
    }

    // Launch Minecraft
    options := launcher.RunOptions{
        LogOutput: func(msg string) {
            fmt.Println(msg)
        },
    }
    
    if err := launcherInstance.Run(false, options); err != nil {
        panic(err)
    }
}
```

### Authentication

```go
import "limeal.fr/launchygo/game/authenticator"

// Microsoft authentication
auth, urlOpts, err := authenticator.FindAuthenticatorFromURI("microsoft://your-code")
if err != nil {
    panic(err)
}

// Custom authentication
auth, urlOpts, err := authenticator.FindAuthenticatorFromURI("custom://user:pass@api.example.com/login")

// Authenticate profile
if err := profile.AuthenticateWithCode(auth, urlOpts.User.Username()); err != nil {
    panic(err)
}
```

### Custom Connectors

The library supports multiple connector types:

- **File Connector**: `file:///path/to/pack`
- **SFTP Connector**: `sftp://user:pass@host:port/path`
- **HTTP Connector**: `http://server.com/path` or `https://server.com/path` (read only)

### Game Folder Structure

A game folder contains:
- `manifest.json`: Pack metadata and file list
- `minecraft.jar`: Main Minecraft JAR file
- `libraries/`: Required libraries
- `natives/`: Native libraries for your platform
- `assets/`: Game assets (textures, sounds, etc.)

## Advanced Features

### Memory Management

```go
profile := profile.NewGameProfile()
profile.SetMemory(8, 4) // 8GB max, 4GB initial memory
```

### Custom Java Path

```go
launcherInstance := launcher.NewLauncher()
launcherInstance.SetJavaPath("/path/to/java")
```

### Quick Play Support

```go
options := launcher.RunOptions{
    GameFeatures: []rules.Feature{
        {
            AKey:  "has_quick_plays_support",
            Flag:  "quickPlayPath",
            Value: "/path/to/quickPlay/log.json",
        },
        {
            AKey:  "is_quick_play_multiplayer",
            Flag:  "quickPlayMultiplayer",
            Value: "mc.example.com",
        },
    },
}
```

### Progress Tracking

```go
progressCallback := func(operation string, current, total int, currentFile string) {
    fmt.Printf("%s: %d/%d - %s\n", operation, current, total, currentFile)
}

if err := gameFolder.Build(false, progressCallback); err != nil {
    panic(err)
}
```

## Requirements

- Go 1.23.4 or later
- Network access (for downloading game assets)

Note: For now it has been completely tested for 1.20.1

## Supported Platforms

- Windows (x64)
- macOS (x64, ARM64 with Rosetta for older versions)
- Linux (x64)

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [SFTP](https://github.com/pkg/sftp) - SFTP client
- [Crypto](https://golang.org/x/crypto) - Cryptographic functions

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
- Create an issue on GitHub
- Check the documentation
- Review the examples in the `cmd/` directory
