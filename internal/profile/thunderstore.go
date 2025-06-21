package profile

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func getThunderstorePackageWithRetry(fullName, community string) (*ThunderstorePackage, error) {
	cacheKey := fmt.Sprintf("%s-%s", community, fullName)
	cacheMutex.RLock()
	if cached, exists := packageCache[cacheKey]; exists {
		cacheMutex.RUnlock()
		log.Printf("Using cached package info for: %s\n", fullName)
		return cached, nil
	}
	cacheMutex.RUnlock()

	maxRetries := 3
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(int64(baseDelay) * (1 << uint(attempt)))
			if delay > maxDelay {
				delay = maxDelay
			}
			jitter := time.Duration(rand.Int63n(int64(delay / 2)))
			delay = delay + jitter - delay/4

			log.Printf("Rate limited, retrying in %v (attempt %d/%d)\n", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}

		pkg, err := getThunderstorePackage(fullName, community)
		if err == nil {
			cacheMutex.Lock()
			packageCache[cacheKey] = pkg
			cacheMutex.Unlock()
			return pkg, nil
		}

		lastErr = err
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") {
			continue
		}

		return nil, err
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func getThunderstorePackage(fullName, community string) (*ThunderstorePackage, error) {
	parts := strings.SplitN(fullName, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid package name format: %s (expected namespace-name)", fullName)
	}
	namespace := parts[0]
	name := parts[1]

	url := fmt.Sprintf("https://thunderstore.io/c/%s/api/v1/package/", community)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch packages for %s: %w", community, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429) for community %s", community)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch packages for community %s: status %d", community, resp.StatusCode)
	}

	var packages []ThunderstorePackage
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return nil, fmt.Errorf("failed to parse packages response for %s: %w", community, err)
	}

	for _, pkg := range packages {
		if pkg.Owner == namespace && pkg.Name == name {
			if pkg.IsDeprecated {
				log.Printf("Warning: Package %s is deprecated\n", fullName)
			}

			if len(pkg.Versions) > 0 {
				pkg.Latest = &pkg.Versions[0]
			}

			return &pkg, nil
		}
	}

	return nil, fmt.Errorf("package %s not found in community %s", fullName, community)
}

func downloadAndExtractMod(mod ModInfo, pluginsPath, community string) error {
	version := fmt.Sprintf("%d.%d.%d", mod.Version.Major, mod.Version.Minor, mod.Version.Patch)
	return downloadAndInstallMod(mod.Name, version, pluginsPath, community)
}

func downloadAndInstallMod(fullName, version, pluginsPath, community string) error {
	pkg, err := getThunderstorePackageWithRetry(fullName, community)
	if err != nil {
		return fmt.Errorf("failed to get package info for %s: %w", fullName, err)
	}

	if pkg.Latest.VersionNumber != version {
		log.Printf("Warning: Requested version %s not available for %s, using latest %s\n",
			version, fullName, pkg.Latest.VersionNumber)
	}

	log.Printf("Downloading mod: %s v%s\n", fullName, pkg.Latest.VersionNumber)

	packageFile, err := downloadModPackageWithRetry(pkg.Latest.DownloadURL, fullName)
	if err != nil {
		return fmt.Errorf("failed to download package %s: %w", fullName, err)
	}
	defer os.Remove(packageFile.Name())
	defer packageFile.Close()

	return extractModToPlugins(packageFile.Name(), fullName, pluginsPath)
}

func downloadModPackageWithRetry(downloadURL, fullName string) (*os.File, error) {
	maxRetries := 3
	baseDelay := 2 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(int64(baseDelay) * (1 << uint(attempt-1)))
			jitter := time.Duration(rand.Int63n(int64(delay / 4)))
			delay = delay + jitter

			log.Printf("Download failed, retrying in %v (attempt %d/%d)\n", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Get(downloadURL)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			if resp.StatusCode == 429 {
				continue
			}
			return nil, lastErr
		}

		tempFile, err := os.CreateTemp("", fmt.Sprintf("mod_%s_*.zip", strings.ReplaceAll(fullName, "-", "_")))
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}

		_, err = io.Copy(tempFile, resp.Body)
		resp.Body.Close()

		if err != nil {
			tempFile.Close()
			os.Remove(tempFile.Name())
			lastErr = fmt.Errorf("failed to download: %w", err)
			continue
		}

		tempFile.Seek(0, 0)
		return tempFile, nil
	}

	return nil, fmt.Errorf("failed to download %s after %d attempts: %w", fullName, maxRetries, lastErr)
}

func extractModToPlugins(packagePath, fullName, pluginsPath string) error {
	reader, err := zip.OpenReader(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open mod package %s: %w", fullName, err)
	}
	defer reader.Close()

	if strings.Contains(strings.ToLower(fullName), "bepinex") {
		return extractBepInExPack(reader, pluginsPath, fullName)
	}

	log.Printf("Extracting mod: %s\n", fullName)

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}

		fileName := filepath.Base(file.Name)
		if fileName == "manifest.json" || fileName == "icon.png" ||
			fileName == "README.md" || fileName == "CHANGELOG.md" {
			continue
		}

		var outputPath string
		if strings.HasSuffix(strings.ToLower(fileName), ".dll") {
			outputPath = filepath.Join(pluginsPath, fullName, fileName)
		} else {
			outputPath = filepath.Join(pluginsPath, fullName, file.Name)
		}

		if outputPath != "" {
			err := extractFileToPath(file, outputPath)
			if err != nil {
				return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
			}
			log.Printf("Extracted mod file: %s -> %s\n", file.Name, outputPath)
		}
	}

	return nil
}

func extractBepInExPack(reader *zip.ReadCloser, pluginsPath, fullName string) error {
	profilePath := filepath.Dir(filepath.Dir(pluginsPath))
	bepInExPath := filepath.Join(profilePath, "BepInEx")
	corePath := filepath.Join(bepInExPath, "core")

	err := os.MkdirAll(corePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create BepInEx core directory: %w", err)
	}

	log.Printf("Extracting BepInEx pack: %s\n", fullName)
	extractedAnyCore := false

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}

		fileName := filepath.Base(file.Name)
		filePath := file.Name

		if fileName == "manifest.json" || fileName == "icon.png" ||
			fileName == "README.md" || fileName == "CHANGELOG.md" {
			continue
		}

		var outputPath string

		if strings.Contains(filePath, "/core/") {
			parts := strings.Split(filePath, "/core/")
			if len(parts) >= 2 {
				relativePath := parts[len(parts)-1]
				outputPath = filepath.Join(corePath, relativePath)
				extractedAnyCore = true
			}
		} else if strings.Contains(filePath, "BepInExPack/") && strings.Contains(filePath, "/BepInEx/") {
			if strings.Contains(filePath, "/BepInEx/core/") {
				parts := strings.SplitN(filePath, "/BepInEx/core/", 2)
				if len(parts) == 2 {
					outputPath = filepath.Join(corePath, parts[1])
					extractedAnyCore = true
				}
			} else {
				parts := strings.SplitN(filePath, "/BepInEx/", 2)
				if len(parts) == 2 {
					outputPath = filepath.Join(bepInExPath, parts[1])
				}
			}
		} else if fileName == "doorstop_config.ini" || fileName == "winhttp.dll" {
			outputPath = filepath.Join(profilePath, fileName)
		} else if strings.HasSuffix(strings.ToLower(fileName), ".dll") {
			coreFileNames := []string{
				"BepInEx.dll", "BepInEx.Preloader.dll", "BepInEx.Harmony.dll",
				"0Harmony.dll", "0Harmony20.dll", "HarmonyXInterop.dll",
				"Mono.Cecil.dll", "Mono.Cecil.Mdb.dll", "Mono.Cecil.Pdb.dll", "Mono.Cecil.Rocks.dll",
				"MonoMod.RuntimeDetour.dll", "MonoMod.Utils.dll",
			}

			for _, coreFile := range coreFileNames {
				if strings.EqualFold(fileName, coreFile) {
					outputPath = filepath.Join(corePath, fileName)
					extractedAnyCore = true
					break
				}
			}

			if outputPath == "" {
				outputPath = filepath.Join(corePath, fileName)
			}
		} else if strings.HasSuffix(strings.ToLower(fileName), ".xml") {
			outputPath = filepath.Join(corePath, fileName)
		}

		if outputPath != "" {
			err := extractFileToPath(file, outputPath)
			if err != nil {
				return fmt.Errorf("failed to extract BepInEx file %s: %w", file.Name, err)
			}
			log.Printf("Extracted BepInEx file: %s -> %s\n", file.Name, outputPath)
		}
	}

	if !extractedAnyCore {
		log.Printf("Warning: No core files were extracted from BepInEx package %s\n", fullName)
	}

	return nil
}
