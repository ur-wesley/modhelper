package profile

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ur-wesley/modhelper/internal"
	"github.com/ur-wesley/modhelper/internal/config"
)

func DownloadAndInstall(game internal.Game, targetDir string) error {
	if game.URL == "" {
		return fmt.Errorf("no download URL for game %s", game.Name)
	}

	log.Printf("Downloading profile for %s from %s", game.Name, game.URL)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(game.URL)
	if err != nil {
		return fmt.Errorf("failed to download profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read download data: %w", err)
	}

	log.Printf("Downloaded profile for %s (version: %s)", game.Name, game.Version)

	zipReader, err := zip.NewReader(strings.NewReader(string(buf)), int64(len(buf)))
	if err != nil {
		return fmt.Errorf("failed to read ZIP data: %w", err)
	}

	isR2ZFile := false
	for _, f := range zipReader.File {
		if f.Name == "export.r2x" {
			isR2ZFile = true
			break
		}
	}

	profileDir := config.GetGameProfileDir(game)

	if isR2ZFile {
		log.Printf("Detected r2z file, processing with mod installation...")

		tempFile, err := os.CreateTemp("", "profile_*.r2z")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		_, err = tempFile.Write(buf)
		if err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tempFile.Close()

		err = extractAndInstallR2Z(tempFile.Name(), game, profileDir)
		if err != nil {
			return fmt.Errorf("failed to install r2z profile: %w", err)
		}
	} else {
		log.Printf("Processing as regular ZIP file...")

		fullProfilePath := filepath.Join(profileDir, game.ProfileName)

		err = os.MkdirAll(fullProfilePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create profile directory: %w", err)
		}

		log.Printf("Installing profile to: %s", fullProfilePath)

		for _, f := range zipReader.File {
			if f.FileInfo().IsDir() {
				continue
			}

			destPath := filepath.Join(fullProfilePath, f.Name)

			destDir := filepath.Dir(destPath)
			err = os.MkdirAll(destDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destDir, err)
			}

			err := extractFileFromZip(f, destPath)
			if err != nil {
				return fmt.Errorf("failed to extract %s: %w", f.Name, err)
			}
		}

		bepInExPath := filepath.Join(fullProfilePath, "BepInEx")
		cachePath := filepath.Join(bepInExPath, "cache")
		if err := os.RemoveAll(cachePath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to clean cache directory: %v", err)
		}

		logPath := filepath.Join(bepInExPath, "LogOutput.log")
		if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
			log.Printf("Note: Could not remove log file: %v", err)
		}
	}

	err = SaveProfileVersion(game, targetDir)
	if err != nil {
		log.Printf("Warning: Failed to save profile version file for %s: %v", game.Name, err)
	}

	err = SaveProfileVersionInModsYML(game, targetDir)
	if err != nil {
		log.Printf("Warning: Failed to save profile version in mods.yml for %s: %v", game.Name, err)
	}

	log.Printf("Successfully installed profile for %s", game.Name)
	return nil
}

func extractAndInstallR2Z(r2zPath string, game internal.Game, targetDir string) error {
	profileName := getProfileName(game)
	profilePath := filepath.Join(targetDir, profileName)

	log.Printf("Processing r2z file for profile: %s\n", profilePath)

	reader, err := zip.OpenReader(r2zPath)
	if err != nil {
		return fmt.Errorf("failed to open r2z file: %w", err)
	}
	defer reader.Close()

	err = os.MkdirAll(profilePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	var exportR2X *ExportFormatR2X
	for _, file := range reader.File {
		if file.Name == "export.r2x" {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open export.r2x: %w", err)
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return fmt.Errorf("failed to read export.r2x: %w", err)
			}

			err = yaml.Unmarshal(data, &exportR2X)
			if err != nil {
				return fmt.Errorf("failed to parse export.r2x: %w", err)
			}
			break
		}
	}

	if exportR2X == nil {
		return fmt.Errorf("export.r2x not found in r2z file")
	}

	bepInExPath := filepath.Join(profilePath, "BepInEx")
	pluginsPath := filepath.Join(bepInExPath, "plugins")
	configPath := filepath.Join(bepInExPath, "config")
	corePath := filepath.Join(bepInExPath, "core")

	for _, dir := range []string{bepInExPath, pluginsPath, configPath, corePath} {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if file.Name == "export.r2x" {
			continue
		}

		destPath := filepath.Join(profilePath, file.Name)

		destDir := filepath.Dir(destPath)
		err = os.MkdirAll(destDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		err = extractFileFromZip(file, destPath)
		if err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}

	log.Println("Cleaning BepInEx cache to prevent startup issues...")
	cachePath := filepath.Join(bepInExPath, "cache")
	if err := os.RemoveAll(cachePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Failed to clean cache directory: %v\n", err)
	}

	logPath := filepath.Join(bepInExPath, "LogOutput.log")
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Note: Could not remove log file: %v\n", err)
	}

	statePath := filepath.Join(profilePath, "_state")
	err = os.MkdirAll(statePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create _state directory: %w", err)
	}

	stateFilePath := filepath.Join(statePath, "installation_state.yml")
	stateContent := "currentState: []\n"
	err = os.WriteFile(stateFilePath, []byte(stateContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create installation_state.yml: %w", err)
	}

	log.Println("Downloading and installing mods from Thunderstore...")

	if game.Community == "" {
		return fmt.Errorf("no community found for game %s", game.Name)
	}

	community := game.Community

	err = downloadAndInstallModsCompatible(exportR2X, pluginsPath, community, profilePath)
	if err != nil {
		return fmt.Errorf("failed to download and install mods: %w", err)
	}

	err = createModsYMLFromExport(exportR2X, profilePath)
	if err != nil {
		return fmt.Errorf("failed to create mods.yml: %w", err)
	}

	winhttpPath := filepath.Join(profilePath, "winhttp.dll")
	if _, err := os.Stat(winhttpPath); os.IsNotExist(err) {
		err = os.WriteFile(winhttpPath, []byte{}, 0644)
		if err != nil {
			log.Printf("Warning: Could not create winhttp.dll placeholder: %v\n", err)
		}
	}

	log.Printf("Successfully installed profile: %s\n", profileName)
	return nil
}

func downloadAndInstallModsCompatible(exportR2X *ExportFormatR2X, pluginsPath, community, profilePath string) error {
	if exportR2X == nil {
		return fmt.Errorf("export format is nil")
	}

	enabledMods := make(map[string]bool)
	for _, mod := range exportR2X.Mods {
		if mod.Enabled {
			modKey := fmt.Sprintf("%s-%d.%d.%d", mod.Name, mod.Version.Major, mod.Version.Minor, mod.Version.Patch)
			enabledMods[modKey] = true
		}
	}

	log.Printf("Installing %d enabled mods...\n", len(enabledMods))

	installedMods := make(map[string]bool)

	for _, mod := range exportR2X.Mods {
		if !mod.Enabled {
			continue
		}

		modKey := fmt.Sprintf("%s-%d.%d.%d", mod.Name, mod.Version.Major, mod.Version.Minor, mod.Version.Patch)

		if installedMods[modKey] {
			continue
		}

		err := installModWithDependencies(mod, pluginsPath, community, installedMods, enabledMods)
		if err != nil {
			log.Printf("Warning: Failed to install mod %s: %v\n", modKey, err)
			continue
		}

		installedMods[modKey] = true
		log.Printf("✓ Installed mod: %s\n", modKey)
	}

	essentialFiles := map[string][]string{
		"BepInEx.Preloader.dll": {
			filepath.Join(profilePath, "BepInEx", "core", "BepInEx.Preloader.dll"),
			filepath.Join(profilePath, "config", "BepInEx.Preloader.dll"),
		},
		"BepInEx.cfg": {
			filepath.Join(profilePath, "config", "BepInEx.cfg"),
			filepath.Join(profilePath, "BepInEx", "config", "BepInEx.cfg"),
			filepath.Join(profilePath, "BepInEx.cfg"),
		},
		"doorstop_config.ini": {
			filepath.Join(profilePath, "doorstop_config.ini"),
			filepath.Join(profilePath, "config", "doorstop_config.ini"),
		},
		"installation_state.yml": {
			filepath.Join(profilePath, "_state", "installation_state.yml"),
		},
	}

	for fileName, possiblePaths := range essentialFiles {
		found := false
		for _, checkPath := range possiblePaths {
			if _, err := os.Stat(checkPath); err == nil {
				log.Printf("✓ Found essential file: %s at %s\n", fileName, checkPath)
				found = true
				break
			}
		}
		if !found {
			log.Printf("✗ Missing essential file: %s (checked: %v)\n", fileName, possiblePaths)
			return fmt.Errorf("essential file missing after installation: %s", fileName)
		}
	}

	log.Printf("✓ Successfully installed %d mods with r2modman compatibility\n", len(installedMods))
	return nil
}

func installModWithDependencies(mod ModInfo, pluginsPath, community string, installedMods, enabledMods map[string]bool) error {
	modKey := fmt.Sprintf("%s-%d.%d.%d", mod.Name, mod.Version.Major, mod.Version.Minor, mod.Version.Patch)

	if installedMods[modKey] {
		return nil
	}

	err := downloadAndExtractMod(mod, pluginsPath, community)
	if err != nil {
		return fmt.Errorf("failed to download mod %s: %w", modKey, err)
	}

	installedMods[modKey] = true
	return nil
}

func createModsYMLFromExport(exportR2X *ExportFormatR2X, profilePath string) error {
	var modsYML ModsYML
	currentTime := time.Now().Unix() * 1000

	for _, mod := range exportR2X.Mods {
		parts := strings.SplitN(mod.Name, "-", 2)
		authorName := mod.Name
		displayName := mod.Name
		if len(parts) == 2 {
			authorName = parts[0]
			displayName = parts[1]
		}

		entry := ModEntry{
			ManifestVersion:      1,
			Name:                 mod.Name,
			AuthorName:           authorName,
			WebsiteURL:           fmt.Sprintf("https://thunderstore.io/c/repo/p/%s/", strings.Replace(mod.Name, "-", "/", 1)),
			DisplayName:          displayName,
			Description:          "Mod installed by ModHelper",
			GameVersion:          "0",
			NetworkMode:          "both",
			PackageType:          "other",
			InstallMode:          "managed",
			InstalledAtTime:      currentTime,
			Loaders:              []string{},
			Dependencies:         []string{},
			Incompatibilities:    []string{},
			OptionalDependencies: []string{},
			VersionNumber: VersionNumber{
				Major: mod.Version.Major,
				Minor: mod.Version.Minor,
				Patch: mod.Version.Patch,
			},
			Enabled: mod.Enabled,
			Icon:    "",
		}
		modsYML = append(modsYML, entry)
	}

	data, err := yaml.Marshal(modsYML)
	if err != nil {
		return fmt.Errorf("failed to marshal mods.yml: %w", err)
	}

	modsYMLPath := filepath.Join(profilePath, "mods.yml")
	err = os.WriteFile(modsYMLPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write mods.yml: %w", err)
	}

	log.Printf("Created mods.yml with %d mods\n", len(modsYML))
	return nil
}
