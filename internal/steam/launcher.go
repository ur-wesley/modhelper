package steam

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/ur-wesley/modhelper/internal"
	"github.com/ur-wesley/modhelper/internal/config"
	"github.com/ur-wesley/modhelper/internal/profile"
)

func IsGameInstalled(game internal.Game, steamApps map[string]App) bool {
	_, exists := steamApps[game.ID]
	return exists
}

func findSteamExe() (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("only Windows supported")
	}

	steamPath, err := getSteamPath()
	if err == nil {
		steamExe := filepath.Join(steamPath, "Steam.exe")
		if _, err := os.Stat(steamExe); err == nil {
			return steamExe, nil
		}
	}

	locations := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Steam", "Steam.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Steam", "Steam.exe"),
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Steam.exe not found in common locations")
}

func launchWithOverlay(appID string, gameArgs []string) error {
	steamExe, err := findSteamExe()
	if err != nil {
		return err
	}

	args := []string{"-applaunch", appID}
	if len(gameArgs) > 0 {
		args = append(args, gameArgs...)
	}

	fmt.Printf("Launching Steam: %s %v\n", steamExe, args)
	cmd := exec.Command(steamExe, args...)
	return cmd.Start()
}

func launchDirectly(game internal.Game, gameArgs []string, steamApps map[string]App) error {
	exePath, err := FindGameExeAuto(game.ID, game.Name)
	if err != nil {
		return fmt.Errorf("failed to find game executable: %w", err)
	}

	fmt.Printf("Launching directly: %s %v\n", exePath, gameArgs)
	cmd := exec.Command(exePath, gameArgs...)
	cmd.Dir = filepath.Dir(exePath)
	return cmd.Start()
}

func LaunchGame(game internal.Game, targetDir string, steamApps map[string]App) error {
	_, exists := steamApps[game.ID]
	if !exists {
		return fmt.Errorf("game not installed: %s (ID: %s)", game.Name, game.ID)
	}

	profileInstalled := profile.IsInstalled(game, targetDir)
	var gameArgs []string
	if profileInstalled && game.LaunchArgs != "" {
		gameProfileDir := config.GetGameProfileDir(game)
		profileName := profile.GetActualProfileName(game)
		if profileName == "" {
			profileName = "Default"
		}

		log.Printf("=== Launch Debug Info for %s ===", game.Name)
		log.Printf("Game Profile Dir: %s", gameProfileDir)
		log.Printf("Profile Name from manifest: '%s'", profileName)
		log.Printf("Profile Name from game object: '%s'", game.ProfileName)

		fullProfilePath := filepath.Join(gameProfileDir, profileName)
		log.Printf("Full profile path: %s", fullProfilePath)
		if _, err := os.Stat(fullProfilePath); os.IsNotExist(err) {
			log.Printf("WARNING: Profile directory does not exist: %s", fullProfilePath)

			if entries, err := os.ReadDir(gameProfileDir); err == nil {
				log.Printf("Available profiles in %s:", gameProfileDir)
				for _, entry := range entries {
					if entry.IsDir() {
						log.Printf("  - %s", entry.Name())
					}
				}
			}
		}

		launchArgs := game.LaunchArgs
		launchArgs = strings.ReplaceAll(launchArgs, "${profileLoc}", gameProfileDir)
		launchArgs = strings.ReplaceAll(launchArgs, "${profileName}", profileName)
		launchArgs = strings.ReplaceAll(launchArgs, "/", "\\")

		log.Printf("Final launch args: %s", launchArgs)

		gameArgs = parseArguments(launchArgs)
	} else {
		log.Printf("Launching %s without profile via Steam", game.Name)
	}

	return launchWithOverlay(game.ID, gameArgs)
}

func findGameExecutable(gamePath, gameName string) (string, error) {
	fmt.Printf("Looking for executable in: %s\n", gamePath)

	if _, err := os.Stat(gamePath); os.IsNotExist(err) {
		return "", fmt.Errorf("game directory does not exist: %s", gamePath)
	}

	patterns := []string{
		gameName + ".exe",
		strings.ReplaceAll(gameName, " ", "") + ".exe",
		strings.ReplaceAll(gameName, " ", "_") + ".exe",
		strings.ReplaceAll(gameName, ".", "") + ".exe",
	}

	switch gameName {
	case "Lethal Company":
		patterns = append(patterns, "LethalCompany.exe")
	case "R.E.P.O.":
		patterns = append(patterns, "R.E.P.O.exe", "REPO.exe")
	}

	fmt.Printf("Trying patterns: %v\n", patterns)

	for _, pattern := range patterns {
		exePath := filepath.Join(gamePath, pattern)
		if _, err := os.Stat(exePath); err == nil {
			return exePath, nil
		}
	}

	files, err := os.ReadDir(gamePath)
	if err != nil {
		return "", fmt.Errorf("cannot read game directory: %w", err)
	}

	fmt.Printf("Scanning directory for .exe files...\n")
	var foundExes []string

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".exe") {
			foundExes = append(foundExes, file.Name())
			name := strings.ToLower(file.Name())
			if strings.Contains(name, "unins") || strings.Contains(name, "setup") ||
				strings.Contains(name, "redist") || strings.Contains(name, "vcredist") {
				continue
			}
			return filepath.Join(gamePath, file.Name()), nil
		}
	}

	if len(foundExes) > 0 {
		return "", fmt.Errorf("no suitable executable found in %s (found: %v)", gamePath, foundExes)
	}

	return "", fmt.Errorf("no executable found in %s", gamePath)
}

func findGameExecutableAuto(gamePath, gameName, gameID string) (string, error) {
	fmt.Printf("Looking for executable in: %s\n", gamePath)

	if _, err := os.Stat(gamePath); os.IsNotExist(err) {
		return "", fmt.Errorf("game directory does not exist: %s", gamePath)
	}

	patterns := []string{
		gameName + ".exe",
		strings.ReplaceAll(gameName, " ", "") + ".exe",
		strings.ReplaceAll(gameName, " ", "_") + ".exe",
		strings.ReplaceAll(gameName, ".", "") + ".exe",
	}


	fmt.Printf("Trying patterns: %v\n", patterns)

	for _, pattern := range patterns {
		exePath := filepath.Join(gamePath, pattern)
		if _, err := os.Stat(exePath); err == nil {
			return exePath, nil
		}
	}

	files, err := os.ReadDir(gamePath)
	if err != nil {
		return "", fmt.Errorf("cannot read game directory: %w", err)
	}

	fmt.Printf("Scanning directory for .exe files...\n")
	var foundExes []string

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".exe") {
			foundExes = append(foundExes, file.Name())
			name := strings.ToLower(file.Name())
			if strings.Contains(name, "unins") || strings.Contains(name, "setup") ||
				strings.Contains(name, "redist") || strings.Contains(name, "vcredist") {
				continue
			}
			return filepath.Join(gamePath, file.Name()), nil
		}
	}

	if len(foundExes) > 0 {
		return "", fmt.Errorf("no suitable executable found in %s (found: %v)", gamePath, foundExes)
	}

	return "", fmt.Errorf("no executable found in %s", gamePath)
}

func FindGameExeAuto(appID string, gameName string) (string, error) {
	steamPath, err := getSteamPath()
	if err != nil {
		return "", err
	}

	libs := []string{filepath.Join(steamPath, "steamapps")}
	libFile := filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")
	if data, err := os.ReadFile(libFile); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, `"path"`) {
				parts := strings.Split(line, `"`)
				if len(parts) >= 4 {
					libs = append(libs, filepath.Join(parts[3], "steamapps"))
				}
			}
		}
	}

	manifestName := fmt.Sprintf("appmanifest_%s.acf", appID)
	for _, lib := range libs {
		manifest := filepath.Join(lib, manifestName)
		if _, err := os.Stat(manifest); err == nil {
			data, err := os.ReadFile(manifest)
			if err != nil {
				continue
			}

			re := regexp.MustCompile(`"installdir"\s+"(.+)"`)
			matches := re.FindStringSubmatch(string(data))
			if len(matches) < 2 {
				continue
			}

			installDir := matches[1]
			gameDir := filepath.Join(lib, "common", installDir)

			fmt.Printf("Found game directory: %s\n", gameDir)

			patterns := []string{
				gameName + ".exe",
				strings.ReplaceAll(gameName, " ", "") + ".exe",
				strings.ReplaceAll(gameName, ".", "") + ".exe",
			}

			fmt.Printf("Trying executable patterns: %v\n", patterns)

			for _, pattern := range patterns {
				exePath := filepath.Join(gameDir, pattern)
				if _, err := os.Stat(exePath); err == nil {
					fmt.Printf("Found executable: %s\n", exePath)
					return exePath, nil
				}
			}

			files, err := os.ReadDir(gameDir)
			if err != nil {
				continue
			}

			fmt.Printf("Scanning directory for executables...\n")
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".exe") {
					name := strings.ToLower(file.Name())
					if strings.Contains(name, "unins") || strings.Contains(name, "setup") ||
						strings.Contains(name, "redist") || strings.Contains(name, "vcredist") {
						continue
					}
					exePath := filepath.Join(gameDir, file.Name())
					fmt.Printf("Found fallback executable: %s\n", exePath)
					return exePath, nil
				}
			}
		}
	}
	return "", fmt.Errorf("game %s (ID: %s) executable not found in any Steam library", gameName, appID)
}

func parseArguments(cmdLine string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for _, char := range cmdLine {
		switch char {
		case '"':
			if inQuotes {
				inQuotes = false
			} else {
				inQuotes = true
			}
		case ' ', '\t':
			if inQuotes {
				current.WriteRune(char)
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
