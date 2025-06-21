package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ur-wesley/modhelper/internal/config"
	"github.com/ur-wesley/modhelper/internal/updater"
	"github.com/ur-wesley/modhelper/ui"
)

func main() {
	setupFileLogging()

	updater.CleanupUpdateFiles()

	adminMode := flag.Bool("admin", false, "Run in admin mode for configuration")
	flag.Parse()

	log.Printf("Starting %s %s", AppName, AppVersion)

	if *adminMode {
		log.Println("Running in admin mode")
		ui.RunAdmin()
	} else {
		log.Println("Running in user mode")

		cfg, err := config.Load()
		if err != nil {
			log.Printf("Failed to load config: %v", err)
			return
		}

		ui.ShowUserInterface(cfg)
	}
}

func setupFileLogging() {
	if isPackagedFyneApp() {
		log.SetFlags(log.LstdFlags)
		return
	}

	logsDir := "logs"
	err := os.MkdirAll(logsDir, 0755)
	if err != nil {
		fmt.Printf("Warning: Could not create logs directory: %v\n", err)
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFileName := fmt.Sprintf("modhelper_%s.log", timestamp)
	logFilePath := filepath.Join(logsDir, logFileName)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Warning: Could not create log file: %v\n", err)
		return
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Printf("Logging to file: %s\n", logFilePath)
}

func isPackagedFyneApp() bool {
	execPath, err := os.Executable()
	if err != nil {
		return true
	}

	execDir := filepath.Dir(execPath)

	if _, err := os.Stat(filepath.Join(execDir, "main.go")); err == nil {
		return false
	}

	if _, err := os.Stat(filepath.Join(execDir, "go.mod")); err == nil {
		return false
	}

	execName := filepath.Base(execPath)
	developmentNames := []string{"main.exe", "main", "go", "__debug_bin"}
	for _, devName := range developmentNames {
		if execName == devName {
			return false
		}
	}

	return true
}
