package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"limeal.fr/launchygo/game/folder"
	"limeal.fr/launchygo/game/folder/rules"
	"limeal.fr/launchygo/game/profile"
)

type Launcher struct {
	gameFolder *folder.GameFolder
	profile    *profile.GameProfile

	JavaPath      string   `json:"javaPath"`
	ExtraJavaArgs []string `json:"javaArgs"`
}

func NewLauncher() *Launcher {
	return &Launcher{
		gameFolder:    nil,
		profile:       nil,
		JavaPath:      "",
		ExtraJavaArgs: []string{},
	}
}

func (g *Launcher) SetGameFolder(gameFolder *folder.GameFolder) error {
	g.gameFolder = gameFolder

	if g.JavaPath == "" {
		r, err := g.gameFolder.GetRuntime()
		if err == nil {
			g.JavaPath = r
		} else {
			javaVersion := GetJavaVersionForVersion(g.gameFolder.GetMCVersion())
			javaPath, err := GetJavaPath(javaVersion, "")
			if err != nil {
				return fmt.Errorf("failed to get java path for java_version=%s, mc_version=%s", javaVersion, g.gameFolder.GetMCVersion())
			}
			g.JavaPath = javaPath
		}
	}

	return nil
}

func (g *Launcher) SetProfile(profile *profile.GameProfile) {
	g.profile = profile
}

func (g *Launcher) SetJavaPath(javaPath string) {
	g.JavaPath = javaPath
}

func (g *Launcher) AddJavaArgs(javaArgs []string) {
	g.ExtraJavaArgs = append(g.ExtraJavaArgs, javaArgs...)
}

type RunOptions struct {
	LogOutput       func(string)
	SeparatedThread bool
	OnProcessExit   func()          // Callback when process exits
	GameFeatures    []rules.Feature // Features to use for the game
}

/////////////////////////////////////////////////////////////////////
// Run
/////////////////////////////////////////////////////////////////////

func (g *Launcher) Run(debug bool, options ...RunOptions) error {

	if g.JavaPath == "" {
		return fmt.Errorf("java executable not found in PATH, please init your game folder")
	}

	// Get options
	var runOptions RunOptions
	if len(options) > 0 {
		runOptions = options[0]
	}

	// Helper function to log if available
	log := func(msg string) {
		if runOptions.LogOutput != nil {
			runOptions.LogOutput(msg)
		}
	}

	var args []string
	version := g.gameFolder.GetMCVersion()

	argumentParser := NewLauncherArgumentParser(g)

	args = argumentParser.parseAndFormatArgs(g.gameFolder.GetArguments().JVM)
	// Append xmx and xms
	args = append(args, fmt.Sprintf("-Xmx%dG", g.profile.Memory.Xmx))
	args = append(args, fmt.Sprintf("-Xms%dG", g.profile.Memory.Xms))

	// Add natives library path
	nativesPath := filepath.Join(g.gameFolder.GetPath(), "natives")
	javaLibPath := fmt.Sprintf("-Djava.library.path=%s", nativesPath)
	if !slices.Contains(args, "-Djava.library.path") {
		args = append(args, javaLibPath)
	}
	lwjglLibPath := fmt.Sprintf("-Dorg.lwjgl.librarypath=%s", nativesPath)
	if !slices.Contains(args, "-Dorg.lwjgl.librarypath") {
		args = append(args, lwjglLibPath)
	}

	// Add sound-related JVM arguments
	args = append(args, "-Dorg.lwjgl.util.Debug=true")

	args = append(args, g.gameFolder.GetMainClass())
	args = append(args, argumentParser.parseAndFormatArgs(g.gameFolder.GetArguments().Game, runOptions.GameFeatures...)...)

	// If version is < 1.19 we need to use the arch64 rosetta for macos
	var maj, min int
	fmt.Sscanf(version, "%d.%d", &maj, &min)

	cmdExec := g.JavaPath
	if VersionLT(version, "1.19") && runtime.GOOS == "darwin" {
		arch, err := exec.LookPath("arch")
		if err != nil {
			return fmt.Errorf("arch executable not found in PATH")
		}

		javaVersion := GetJavaVersionForVersion(g.gameFolder.GetMCVersion())
		javaPath, err := GetJavaPath(javaVersion, "x86_64")
		if err != nil {
			return fmt.Errorf("failed to get java path for java_version=%s, mc_version=%s", javaVersion, g.gameFolder.GetMCVersion())
		}
		g.JavaPath = javaPath

		cmdExec = arch
		args = append([]string{"-x86_64", g.JavaPath}, args...)
		log("\nUsing Rosetta for Java " + g.JavaPath)
	}

	log("Building command line arguments...")
	log("Version: " + g.gameFolder.GetVersion())
	log("Game folder: " + g.gameFolder.GetPath())
	log("Java path: " + g.JavaPath)
	log("Command: " + cmdExec + " " + strings.Join(args, " "))

	fmt.Println("[*] Running Minecraft")
	fmt.Println("- Version:", g.gameFolder.GetVersion())
	fmt.Println("- Game folder:", g.gameFolder.GetPath())
	fmt.Println("- Running with Java:", g.JavaPath)
	fmt.Println("- Running with cmd:", cmdExec)
	fmt.Println("- Arguments:")
	for i, arg := range args {
		fmt.Printf("    [%2d] %s\n", i, arg)
	}

	var cmd *exec.Cmd

	if g.JavaPath == "" {
		return fmt.Errorf("java executable not found in PATH")
	}

	cmd = exec.Command(cmdExec, args...)

	log("Command: " + cmd.String())

	if runOptions.SeparatedThread {
		log("Starting Minecraft process in separated thread...")
		return g.runInSeparatedThread(cmd, log, runOptions)
	} else {
		log("Starting Minecraft process...")
		return g.runInSameThread(cmd, log)
	}
}

func (g *Launcher) runInSameThread(cmd *exec.Cmd, log func(string)) error {
	fmt.Println("Running command:", cmd.String())
	cmd.Dir = g.gameFolder.GetPath()

	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		// Non-zero exit -> *exec.ExitError (still has ProcessState)
		if ee, ok := err.(*exec.ExitError); ok && ee.ProcessState != nil {
			errorMsg := fmt.Sprintf("Minecraft exited with code: %d (error: %v)", ee.ExitCode(), err)
			log("ERROR: " + errorMsg)
			fmt.Printf(errorMsg + "\n")
		} else {
			errorMsg := "Error running Minecraft: " + err.Error()
			log("ERROR: " + errorMsg)
			fmt.Println(errorMsg)
		}
		return err
	}

	if cmd.ProcessState != nil {
		exitMsg := fmt.Sprintf("Minecraft exited with code: %d", cmd.ProcessState.ExitCode())
		log(exitMsg)
		fmt.Println(exitMsg)
	} else {
		log("Minecraft exited (no ProcessState)")
		fmt.Println("Minecraft exited (no ProcessState)")
	}

	log("Game process completed successfully")
	return nil
}

func (g *Launcher) runInSeparatedThread(cmd *exec.Cmd, log func(string), runOptions RunOptions) error {
	fmt.Println("Running command in separated process:", cmd.String())
	cmd.Dir = g.gameFolder.GetPath()

	// Set up OS-specific process attributes
	setupWindowsProcessAttributes(cmd)

	// Capture stdout and stderr for logging
	cmd.Stdout = &logWriter{logFunc: log, prefix: "[GAME] "}
	cmd.Stderr = &logWriter{logFunc: log, prefix: "[ERROR] "}

	// Start the process
	if err := cmd.Start(); err != nil {
		errorMsg := "Error starting Minecraft process: " + err.Error()
		log("ERROR: " + errorMsg)
		fmt.Println(errorMsg)
		return err
	}

	log(fmt.Sprintf("Minecraft process started with PID: %d", cmd.Process.Pid))
	fmt.Printf("Minecraft process started with PID: %d\n", cmd.Process.Pid)

	// Start a goroutine to monitor the process and log when it exits
	go func() {
		err := cmd.Wait()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok && ee.ProcessState != nil {
				log(fmt.Sprintf("[PROCESS] Minecraft exited with code: %d", ee.ExitCode()))
			} else {
				log("[PROCESS] Minecraft process error: " + err.Error())
			}
		} else {
			log("[PROCESS] Minecraft process completed successfully")
		}

		// Destroy the natives library
		os.RemoveAll(filepath.Join(g.gameFolder.GetPath(), "natives"))

		// Call the exit callback if provided
		if runOptions.OnProcessExit != nil {
			log("[PROCESS] Calling exit callback...")
			runOptions.OnProcessExit()
		} else {
			// Default behavior: exit the application
			log("[PROCESS] Closing launcher...")
			os.Exit(0)
		}
	}()

	log("Game process launched successfully in separated thread")
	return nil
}

// logWriter implements io.Writer to capture process output
type logWriter struct {
	logFunc func(string)
	prefix  string
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	if lw.logFunc != nil {
		lines := strings.Split(string(p), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				lw.logFunc(lw.prefix + line)
			}
		}
	}
	return len(p), nil
}
