package steam

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unsafe"

	"github.com/ur-wesley/modhelper/internal"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type App struct {
	AppID string
	Name  string
	Path  string
}

func IsGameRunning(game internal.Game) bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	err = windows.Process32First(snapshot, &pe)
	if err != nil {
		return false
	}

	executablePatterns := getExecutablePatterns(game)

	var lowerPatterns []string
	for _, pattern := range executablePatterns {
		lowerPatterns = append(lowerPatterns, strings.ToLower(pattern))
	}

	for {
		processName := windows.UTF16ToString(pe.ExeFile[:])
		processNameLower := strings.ToLower(processName)

		for _, pattern := range lowerPatterns {
			if processNameLower == pattern {
				return true
			}
		}

		err = windows.Process32Next(snapshot, &pe)
		if err != nil {
			break
		}
	}

	return false
}

func StopGame(game internal.Game) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return fmt.Errorf("failed to create process snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	err = windows.Process32First(snapshot, &pe)
	if err != nil {
		return fmt.Errorf("failed to enumerate processes: %w", err)
	}

	executablePatterns := getExecutablePatterns(game)

	var lowerPatterns []string
	for _, pattern := range executablePatterns {
		lowerPatterns = append(lowerPatterns, strings.ToLower(pattern))
	}

	for {
		processName := windows.UTF16ToString(pe.ExeFile[:])
		processNameLower := strings.ToLower(processName)

		var shouldTerminate bool
		for _, pattern := range lowerPatterns {
			if processNameLower == pattern {
				shouldTerminate = true
				break
			}
		}

		if shouldTerminate {
			handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pe.ProcessID)
			if err != nil {
				return fmt.Errorf("failed to open process %s: %w", processName, err)
			}
			defer windows.CloseHandle(handle)

			err = windows.TerminateProcess(handle, 0)
			if err != nil {
				return fmt.Errorf("failed to terminate process %s: %w", processName, err)
			}

			return nil
		}

		err = windows.Process32Next(snapshot, &pe)
		if err != nil {
			break
		}
	}

	return fmt.Errorf("game process not found")
}

func GetGameStatus(game internal.Game, steamApps map[string]App) string {
	_, exists := steamApps[game.ID]
	if !exists {
		return "not_installed"
	}

	if IsGameRunning(game) {
		return "running"
	}

	return "installed"
}

func getSteamPath() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	path, _, err := k.GetStringValue("SteamPath")
	if err != nil {
		return "", err
	}
	return path, nil
}

func parseInstallDir(manifest string) (string, error) {
	f, err := os.Open(manifest)
	if err != nil {
		return "", err
	}
	defer f.Close()

	re := regexp.MustCompile(`"installdir"\s+"(.+)"`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := re.FindStringSubmatch(scanner.Text()); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("installdir not found in %s", manifest)
}

func findGameExe(appID, exeName string) (string, error) {
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
			dir, err := parseInstallDir(manifest)
			if err != nil {
				continue
			}
			exePath := filepath.Join(lib, "common", dir, exeName)
			if _, err := os.Stat(exePath); err == nil {
				return exePath, nil
			}
		}
	}
	return "", fmt.Errorf("game %s not found in any Steam library", appID)
}

func GetPath() (string, error) {
	return getSteamPath()
}

func GetApps() (map[string]App, error) {
	steamPath, err := getSteamPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get Steam path: %w", err)
	}

	apps := make(map[string]App)

	libs := []string{filepath.Join(steamPath, "steamapps")}
	libFile := filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")

	if data, err := os.ReadFile(libFile); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, `"path"`) {
				parts := strings.Split(line, `"`)
				if len(parts) >= 4 {
					libPath := filepath.Join(parts[3], "steamapps")
					if _, err := os.Stat(libPath); err == nil {
						libs = append(libs, libPath)
					}
				}
			}
		}
	}

	for _, libPath := range libs {
		files, err := filepath.Glob(filepath.Join(libPath, "appmanifest_*.acf"))
		if err != nil {
			continue
		}

		for _, file := range files {
			app, err := parseAppManifest(file)
			if err == nil {
				apps[app.AppID] = app
			}
		}
	}

	return apps, nil
}

func parseAppManifest(manifestPath string) (App, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return App{}, err
	}

	content := string(data)
	var app App

	if re := regexp.MustCompile(`"appid"\s*"(\d+)"`); re != nil {
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			app.AppID = matches[1]
		}
	}

	if re := regexp.MustCompile(`"name"\s*"([^"]+)"`); re != nil {
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			app.Name = matches[1]
		}
	}
	if re := regexp.MustCompile(`"installdir"\s*"([^"]+)"`); re != nil {
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			steamappsDir := filepath.Dir(manifestPath)
			app.Path = filepath.Join(steamappsDir, "common", matches[1])
		}
	}

	if app.AppID == "" || app.Name == "" {
		return App{}, fmt.Errorf("invalid manifest file")
	}

	return app, nil
}

func FindGameExe(appID, exeName string) (string, error) {
	return findGameExe(appID, exeName)
}

func getExecutablePatterns(game internal.Game) []string {
	if len(game.ExecutableNames) > 0 {
		return game.ExecutableNames
	}

	patterns := []string{
		game.Name + ".exe",
		strings.ReplaceAll(game.Name, " ", "") + ".exe",
		strings.ReplaceAll(game.Name, " ", "_") + ".exe",
		strings.ReplaceAll(game.Name, ".", "") + ".exe",
	}

	switch game.Name {
	case "Lethal Company":
		patterns = append(patterns, "Lethal Company.exe", "LethalCompany.exe")
	case "R.E.P.O.":
		patterns = append(patterns, "R.E.P.O.exe", "REPO.exe")
	}

	return patterns
}
