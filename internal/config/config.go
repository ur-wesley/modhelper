package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ur-wesley/modhelper/internal"
)

const (
	DefaultManifestURL  = "https://gist.githubusercontent.com/ur-wesley/8e93a37dc70b7d8161e94fc62df061ee/raw/manifest.json"
	R2ModmanDownloadURL = "https://r2modman.net/download/latest-version/"
	ConfigFileName      = "config.json"
)

func Load() (*internal.Config, error) {
	f, err := os.Open(ConfigFileName)
	if err != nil {
		return &internal.Config{
			ManifestURL: DefaultManifestURL,
			TargetDir:   GetDefaultProfileDir(),
		}, nil
	}
	defer f.Close()

	var c internal.Config
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func Save(c *internal.Config) error {
	f, err := os.Create(ConfigFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}

func GetDefaultProfileDir() string {
	appData := os.Getenv("AppData")
	if appData == "" {
		return filepath.Join(os.Getenv("HOME"), ".r2modmanPlus-local")
	}
	return filepath.Join(appData, "r2modmanPlus-local")
}

func GetGameProfileDir(game internal.Game) string {
	baseDir := GetDefaultProfileDir()

	gameDir := strings.ReplaceAll(game.Name, " ", "")

	gameDir = strings.TrimRight(gameDir, ".")

	profilesDir := filepath.Join(baseDir, gameDir, "profiles")

	err := os.MkdirAll(profilesDir, 0755)
	if err != nil {
		log.Printf("Warning: Could not create profiles directory %s: %v", profilesDir, err)
	}

	return profilesDir
}
