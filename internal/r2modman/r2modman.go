package r2modman

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func GetDefaultPath() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	localAppData := os.Getenv("LocalAppData")
	if localAppData == "" {
		return ""
	}
	return filepath.Join(localAppData, "Programs", "r2modman", "r2modman.exe")
}

func Find() (string, error) {
	exe := GetDefaultPath()
	if exe == "" {
		return "", fmt.Errorf("only Windows is supported")
	}
	if _, err := os.Stat(exe); os.IsNotExist(err) {
		return "", fmt.Errorf("r2modman.exe not found at %s", exe)
	}
	return exe, nil
}

func GetVersion(exe string) (string, error) {
	cmd := exec.Command(exe, "--version")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(out)
	version := ""
	if scanner.Scan() {
		version = scanner.Text()
	}
	cmd.Wait()
	return version, nil
}
