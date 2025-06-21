package profile

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ur-wesley/modhelper/internal"
	"github.com/ur-wesley/modhelper/internal/config"
)

func GetProfileVersion(game internal.Game) string {
	return game.Version
}

func SaveProfileVersion(game internal.Game, targetDir string) error {
	profileDir := config.GetGameProfileDir(game)

	versionFile := filepath.Join(profileDir, game.ProfileName, ".profile_version")

	versionData := ProfileVersion{
		URL:     game.URL,
		Version: game.Version,
	}

	data, err := json.Marshal(versionData)
	if err != nil {
		return fmt.Errorf("failed to marshal version data: %w", err)
	}

	err = os.WriteFile(versionFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	log.Printf("Saved profile version for %s: %s", game.Name, game.Version)
	return nil
}

func GetInstalledProfileVersion(game internal.Game, targetDir string) (string, error) {
	profileDir := config.GetGameProfileDir(game)

	versionFile := filepath.Join(profileDir, game.ProfileName, ".profile_version")

	data, err := os.ReadFile(versionFile)
	if err == nil {
		var versionData ProfileVersion
		err = json.Unmarshal(data, &versionData)
		if err == nil {
			return versionData.Version, nil
		}
		log.Printf("Warning: Could not parse .profile_version for %s: %v", game.Name, err)
	}

	version, err := GetVersionFromModsYML(game, targetDir)
	if err != nil {
		log.Printf("Warning: Could not get version from mods.yml for %s: %v", game.Name, err)
	}
	if version != "" {
		return version, nil
	}

	return "", nil
}

func IsProfileUpToDate(game internal.Game, targetDir string) (bool, error) {
	if game.URL == "" || game.Version == "" {
		return true, nil
	}

	installedVersion, err := GetInstalledProfileVersion(game, targetDir)
	if err != nil {
		log.Printf("Warning: Could not check profile version for %s: %v", game.Name, err)
		return true, nil
	}

	if installedVersion == "" {
		return false, nil
	}

	currentVersion := game.Version
	isUpToDate := installedVersion == currentVersion
	if !isUpToDate {
		log.Printf("Profile update available for %s: %s -> %s", game.Name, installedVersion, currentVersion)
	}

	return isUpToDate, nil
}

func SaveProfileVersionInModsYML(game internal.Game, targetDir string) error {
	profileDir := config.GetGameProfileDir(game)

	modsYMLPath := filepath.Join(profileDir, game.ProfileName, "mods.yml")

	var modsYML ModsYML
	if data, err := os.ReadFile(modsYMLPath); err == nil {
		if err := yaml.Unmarshal(data, &modsYML); err != nil {
			log.Printf("Warning: Could not parse existing mods.yml for %s: %v", game.Name, err)
		}
	}

	var filteredMods ModsYML
	for _, mod := range modsYML {
		if mod.Name != "_ProfileVersion" {
			filteredMods = append(filteredMods, mod)
		}
	}

	versionEntry := ModEntry{
		ManifestVersion:      1,
		Name:                 "_ProfileVersion",
		AuthorName:           "ModHelper",
		WebsiteURL:           "",
		DisplayName:          fmt.Sprintf("Profile Version %s", game.Version),
		Description:          fmt.Sprintf("Profile version marker for %s", game.Name),
		GameVersion:          game.Version,
		NetworkMode:          "none",
		PackageType:          "other",
		InstallMode:          "manual",
		InstalledAtTime:      time.Now().Unix() * 1000,
		Loaders:              []string{},
		Dependencies:         []string{},
		Incompatibilities:    []string{},
		OptionalDependencies: []string{},
		VersionNumber: VersionNumber{
			Major: 1,
			Minor: 0,
			Patch: 0,
		},
		Enabled: false,
		Icon:    "",
	}

	finalMods := append([]ModEntry{versionEntry}, filteredMods...)

	data, err := yaml.Marshal(finalMods)
	if err != nil {
		return fmt.Errorf("failed to marshal mods.yml with version: %w", err)
	}

	err = os.WriteFile(modsYMLPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write mods.yml with version: %w", err)
	}

	log.Printf("Saved profile version %s in mods.yml for %s", game.Version, game.Name)
	return nil
}

func GetVersionFromModsYML(game internal.Game, targetDir string) (string, error) {
	profileDir := config.GetGameProfileDir(game)

	modsYMLPath := filepath.Join(profileDir, game.ProfileName, "mods.yml")

	data, err := os.ReadFile(modsYMLPath)
	if err != nil {
		return "", nil
	}

	var modsYML ModsYML
	err = yaml.Unmarshal(data, &modsYML)
	if err != nil {
		return "", fmt.Errorf("failed to parse mods.yml: %w", err)
	}

	for _, mod := range modsYML {
		if mod.Name == "_ProfileVersion" {
			return mod.GameVersion, nil
		}
	}

	return "", nil
}
