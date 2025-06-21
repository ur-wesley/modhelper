package profile

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ur-wesley/modhelper/internal"
	"github.com/ur-wesley/modhelper/internal/config"
)

func IsInstalled(game internal.Game, targetDir string) bool {
	gameProfileDir := config.GetGameProfileDir(game)
	profileName := getProfileName(game)
	profilePath := filepath.Join(gameProfileDir, profileName)

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return false
	}

	modsYmlPath := filepath.Join(profilePath, "mods.yml")
	bepInExPath := filepath.Join(profilePath, "BepInEx")

	if _, err := os.Stat(modsYmlPath); err == nil {
		return true
	}
	if _, err := os.Stat(bepInExPath); err == nil {
		return true
	}

	return false
}

func getProfileName(game internal.Game) string {
	if game.ProfileName != "" {
		return game.ProfileName
	}
	return "Default"
}

func GetActualProfileName(game internal.Game) string {
	gameProfileDir := config.GetGameProfileDir(game)
	manifestProfileName := getProfileName(game)

	fullPath := filepath.Join(gameProfileDir, manifestProfileName)
	if _, err := os.Stat(fullPath); err == nil {
		return manifestProfileName
	}

	variations := []string{
		manifestProfileName,
		strings.TrimRight(manifestProfileName, "."),
		strings.ReplaceAll(manifestProfileName, ".", ""),
		game.Name,
		"Default",
	}

	for _, variation := range variations {
		testPath := filepath.Join(gameProfileDir, variation)
		if _, err := os.Stat(testPath); err == nil {
			log.Printf("Found actual profile directory: '%s' (manifest had: '%s')", variation, manifestProfileName)
			return variation
		}
	}

	return manifestProfileName
}

func GetProfileStatus(game internal.Game, targetDir string) ProfileStatus {
	status := ProfileStatus{}

	status.Installed = IsInstalled(game, targetDir)

	if !status.Installed {
		return status
	}

	upToDate, err := IsProfileUpToDate(game, targetDir)
	if err != nil {
		status.VersionError = err
		status.UpToDate = true
	} else {
		status.UpToDate = upToDate
		status.HasUpdate = !upToDate

		installedVersion, _ := GetInstalledProfileVersion(game, targetDir)
		log.Printf("Version check for %s: installed='%s', manifest='%s', upToDate=%v, hasUpdate=%v",
			game.Name, installedVersion, game.Version, upToDate, status.HasUpdate)
	}

	return status
}

func DeleteProfile(game internal.Game) error {
	profileDir := config.GetGameProfileDir(game)
	fullProfilePath := filepath.Join(profileDir, game.ProfileName)

	if _, err := os.Stat(fullProfilePath); os.IsNotExist(err) {
		return nil
	}

	log.Printf("Deleting profile directory: %s", fullProfilePath)
	err := os.RemoveAll(fullProfilePath)
	if err != nil {
		return fmt.Errorf("failed to delete profile directory: %w", err)
	}

	log.Printf("Successfully deleted profile for %s", game.Name)
	return nil
}
