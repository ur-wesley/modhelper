package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ur-wesley/modhelper/internal"
)

const (
	GitHubAPI    = "https://api.github.com/repos/ur-wesley/modhelper/releases/latest"
	UpdaterName  = "updater_temp.exe"
	BackupSuffix = ".backup"
)

type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	Assets      []GitHubAsset `json:"assets"`
	PublishedAt time.Time     `json:"published_at"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	DownloadURL    string
	ReleaseNotes   string
	Size           int64
}

func CheckForUpdates() (*UpdateInfo, error) {
	log.Println("Checking for updates...")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(GitHubAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	if release.Draft || release.Prerelease {
		log.Println("Latest release is draft or prerelease, skipping")
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: internal.AppVersion,
			LatestVersion:  release.TagName,
		}, nil
	}

	updateInfo := &UpdateInfo{
		CurrentVersion: internal.AppVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
	}

	if !isNewerVersion(release.TagName, internal.AppVersion) {
		log.Printf("Current version %s is up to date (latest: %s)", internal.AppVersion, release.TagName)
		updateInfo.Available = false
		return updateInfo, nil
	}

	var downloadURL string
	var size int64
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".exe") &&
			(strings.Contains(asset.Name, "ModHelper") || strings.Contains(asset.Name, "modhelper")) {
			downloadURL = asset.BrowserDownloadURL
			size = asset.Size
			break
		}
	}

	if downloadURL == "" {
		return nil, fmt.Errorf("no Windows executable found in latest release")
	}

	updateInfo.Available = true
	updateInfo.DownloadURL = downloadURL
	updateInfo.Size = size

	log.Printf("Update available: %s -> %s", internal.AppVersion, release.TagName)
	return updateInfo, nil
}

func isNewerVersion(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for len(latestParts) < 3 {
		latestParts = append(latestParts, "0")
	}
	for len(currentParts) < 3 {
		currentParts = append(currentParts, "0")
	}

	for i := 0; i < 3; i++ {
		var latestNum, currentNum int
		fmt.Sscanf(latestParts[i], "%d", &latestNum)
		fmt.Sscanf(currentParts[i], "%d", &currentNum)

		if latestNum > currentNum {
			return true
		} else if latestNum < currentNum {
			return false
		}
	}

	return false
}

func DownloadUpdate(downloadURL string, progressCallback func(downloaded, total int64)) (string, error) {
	log.Printf("Downloading update from: %s", downloadURL)

	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "modhelper_update_*.exe")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	totalSize := resp.ContentLength
	var downloaded int64

	buffer := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := tempFile.Write(buffer[:n]); writeErr != nil {
				return "", fmt.Errorf("failed to write to temp file: %w", writeErr)
			}
			downloaded += int64(n)

			if progressCallback != nil {
				progressCallback(downloaded, totalSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read download: %w", err)
		}
	}

	log.Printf("Download completed: %s (%d bytes)", tempFile.Name(), downloaded)
	return tempFile.Name(), nil
}

func ApplyUpdate(newExecutablePath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("self-update only supported on Windows")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	log.Printf("Applying update: %s -> %s", newExecutablePath, currentExe)

	backupPath := currentExe + BackupSuffix
	if err := copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	updaterScript := createUpdaterScript(newExecutablePath, currentExe, backupPath)

	scriptPath := filepath.Join(filepath.Dir(currentExe), "update.bat")
	if err := os.WriteFile(scriptPath, []byte(updaterScript), 0755); err != nil {
		return fmt.Errorf("failed to create updater script: %w", err)
	}

	log.Println("Starting update process...")

	cmd := exec.Command("cmd", "/C", scriptPath)
	cmd.Dir = filepath.Dir(currentExe)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start updater: %w", err)
	}

	log.Println("Exiting for update...")
	os.Exit(0)

	return nil
}

func createUpdaterScript(newExe, currentExe, backupPath string) string {
	return fmt.Sprintf(`@echo off
echo Updating ModHelper...

REM Wait for main process to exit
timeout /t 2 /nobreak >nul

REM Replace the executable
copy /Y "%s" "%s"
if errorlevel 1 (
    echo Update failed, restoring backup...
    copy /Y "%s" "%s"
    pause
    exit /b 1
)

REM Clean up
del "%s" 2>nul
del "%s" 2>nul

REM Start the updated application
start "" "%s"

REM Clean up this script
del "%%~f0"
`, newExe, currentExe, backupPath, currentExe, newExe, backupPath, currentExe)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func CleanupUpdateFiles() {
	currentExe, err := os.Executable()
	if err != nil {
		return
	}

	dir := filepath.Dir(currentExe)

	backupPath := currentExe + BackupSuffix
	if _, err := os.Stat(backupPath); err == nil {
		os.Remove(backupPath)
		log.Println("Cleaned up backup file")
	}

	files, err := filepath.Glob(filepath.Join(dir, "modhelper_update_*.exe"))
	if err == nil {
		for _, file := range files {
			os.Remove(file)
		}
	}

	scriptPath := filepath.Join(dir, "update.bat")
	if _, err := os.Stat(scriptPath); err == nil {
		os.Remove(scriptPath)
	}
}
