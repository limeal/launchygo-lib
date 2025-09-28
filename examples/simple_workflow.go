package main

import (
	"fmt"
	"log"
	"os"

	"limeal.fr/launchygo/pkg/connectors"
	"limeal.fr/launchygo/pkg/game/folder"
	"limeal.fr/launchygo/pkg/game/folder/generator"
	"limeal.fr/launchygo/pkg/game/launcher"
	"limeal.fr/launchygo/pkg/game/profile"
)

func main() {
	fmt.Println("🚀 LaunchyGo Simple Workflow: Generate -> Publish -> Launch")
	fmt.Println("=========================================================")
	fmt.Println()

	// Configuration
	packName := "my-minecraft-pack"
	version := "1.20.1"
	serverDir := "./server-packs"

	// Step 1: Generate a Minecraft pack
	fmt.Println("📦 Step 1: Generating Minecraft Pack")
	fmt.Println("-----------------------------------")
	fmt.Printf("Creating Vanilla pack '%s' for version %s...\n", packName, version)

	vanillaGen := generator.InitVanillaGenerator(packName, version)
	vanillaGen.Generate(false, progressCallback)
	fmt.Println("✅ Pack generated successfully!")
	fmt.Println()

	// Step 2: Publish the pack
	fmt.Println("📤 Step 2: Publishing Pack")
	fmt.Println("-------------------------")

	// Create server directory
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		log.Fatal("Failed to create server directory:", err)
	}

	// Connect to server and publish
	serverConnector := connectors.FindConnectorFromURI("file://" + serverDir)
	if err := serverConnector.Connect(); err != nil {
		log.Fatal("Failed to connect to server:", err)
	}
	defer serverConnector.Close()

	fmt.Printf("Publishing pack to: %s\n", serverDir)
	folder.PublishGameFolder(serverConnector, packName)
	fmt.Println("✅ Pack published successfully!")
	fmt.Println()

	// Step 3: Launch Minecraft from published pack
	fmt.Println("🎮 Step 3: Launching Minecraft")
	fmt.Println("-----------------------------")

	// Connect to published pack
	publishedPackURI := "file://" + serverDir
	launchConnector := connectors.FindConnectorFromURI(publishedPackURI)
	if err := launchConnector.Connect(); err != nil {
		log.Fatal("Failed to connect to published pack:", err)
	}
	defer launchConnector.Close()

	// Initialize game folder
	gameFolder, err := folder.InitGameFolder(launchConnector, packName)
	if err != nil {
		log.Fatal("Failed to initialize game folder:", err)
	}

	// Create game profile
	gameProfile := profile.NewGameProfile()
	gameProfile.SetUser("Steve")
	gameProfile.SetMemory(4, 2) // 4GB max, 2GB initial

	// Create and configure launcher
	launcherInstance := launcher.NewLauncher()
	launcherInstance.SetGameFolder(gameFolder)
	launcherInstance.SetProfile(gameProfile)

	// Build game folder
	fmt.Println("Building game folder...")
	if err := gameFolder.Build(false, progressCallback); err != nil {
		log.Fatal("Failed to build game folder:", err)
	}
	fmt.Println("✅ Game folder built successfully!")

	// Launch Minecraft
	fmt.Println("🚀 Launching Minecraft...")
	fmt.Printf("  Version: %s\n", gameFolder.GetVersion())
	fmt.Printf("  Username: %s\n", gameProfile.Username)
	fmt.Printf("  Memory: %dGB max, %dGB initial\n", gameProfile.Memory.Xmx, gameProfile.Memory.Xms)
	fmt.Println()

	launchOptions := launcher.RunOptions{
		LogOutput: func(msg string) {
			fmt.Printf("[GAME] %s\n", msg)
		},
	}

	if err := launcherInstance.Run(false, launchOptions); err != nil {
		log.Fatal("Failed to launch Minecraft:", err)
	}

	fmt.Println("✅ Minecraft launched successfully!")
}

// Progress callback for displaying download progress
func progressCallback(operation string, current, total int, description string) {
	if total > 0 {
		percentage := float64(current) / float64(total) * 100
		fmt.Printf("\r  %s: %d/%d (%.1f%%) - %s", operation, current, total, percentage, description)
	} else {
		fmt.Printf("\r  %s: %s", operation, description)
	}
}
