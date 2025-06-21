package profile

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func extractFileToPath(file *zip.File, outputPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	dir := filepath.Dir(outputPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	if strings.HasSuffix(file.Name, "/") {
		return nil
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

func extractFileFromZip(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

func extractConfigFiles(reader *zip.ReadCloser, configPath string) error {
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "config/") {
			relativePath := strings.TrimPrefix(file.Name, "config/")
			if relativePath == "" || strings.HasSuffix(relativePath, "/") {
				continue
			}

			outputPath := filepath.Join(configPath, relativePath)
			err := extractFileToPath(file, outputPath)
			if err != nil {
				return fmt.Errorf("failed to extract config file %s: %w", file.Name, err)
			}
			log.Printf("Extracted config: %s\n", relativePath)
		}
	}
	return nil
}

func extractOtherFiles(reader *zip.ReadCloser, profilePath string) error {
	for _, file := range reader.File {
		fileName := strings.ToLower(file.Name)

		if strings.HasSuffix(file.Name, "/") ||
			strings.HasPrefix(file.Name, "config/") ||
			strings.HasPrefix(file.Name, "bepinex/") ||
			fileName == "export.r2x" {
			continue
		}

		outputPath := filepath.Join(profilePath, file.Name)
		err := extractFileToPath(file, outputPath)
		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}
		log.Printf("Extracted file: %s\n", file.Name)
	}
	return nil
}
