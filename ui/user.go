package ui

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ur-wesley/modhelper/internal"
	"github.com/ur-wesley/modhelper/internal/profile"
	"github.com/ur-wesley/modhelper/internal/r2modman"
	"github.com/ur-wesley/modhelper/internal/steam"
	"github.com/ur-wesley/modhelper/internal/updater"
)

type GameListItem struct {
	Game       internal.Game
	Container  *fyne.Container
	UpdateFunc func()
}

func GetWindowDimensions() (float32, float32) {
	return float32(internal.WindowWidth), float32(internal.WindowHeight)
}

func fuzzyMatch(target, searchText string) bool {
	if searchText == "" {
		return true
	}

	searchIndex := 0
	searchRunes := []rune(searchText)
	targetRunes := []rune(target)

	for _, targetChar := range targetRunes {
		if searchIndex < len(searchRunes) && searchRunes[searchIndex] == targetChar {
			searchIndex++
			if searchIndex == len(searchRunes) {
				return true
			}
		}
	}

	return searchIndex == len(searchRunes)
}

func ShowUserInterface(cfg *internal.Config) {
	a := app.NewWithID("com.urwesley.modhelper")

	messages := internal.German()

	windowWidth, windowHeight := GetWindowDimensions()
	w := a.NewWindow(fmt.Sprintf("%s %s", internal.AppName, internal.AppVersion))
	w.Resize(fyne.NewSize(windowWidth, windowHeight))

	infoButton := widget.NewButtonWithIcon("", theme.HelpIcon(), func() {
		showInfoDialog(w, messages)
	})
	infoButton.Resize(fyne.NewSize(32, 32))

	updateButton := widget.NewButtonWithIcon("", theme.DownloadIcon(), func() {
		showUpdateDialog(w, messages)
	})
	updateButton.Resize(fyne.NewSize(32, 32))
	updateButton.Hide()

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder(messages.SearchGames)

	gameList := container.NewVBox()

	loadingIcon := widget.NewIcon(theme.ViewRefreshIcon())
	loadingLabel := widget.NewLabel(messages.LoadingGames)
	loadingLabel.Alignment = fyne.TextAlignCenter
	loadingContent := container.NewVBox(
		container.NewCenter(loadingIcon),
		loadingLabel,
	)

	content := container.NewVBox(loadingContent)

	r2modmanBadge := widget.NewLabel("❌ " + messages.R2ModmanStatus)
	steamBadge := widget.NewLabel("❌ " + messages.SteamStatus)
	manifestBadge := widget.NewLabel("❌ " + messages.ManifestStatus)

	footer := container.NewHBox(
		widget.NewLabel("Status:"),
		layout.NewSpacer(),
		r2modmanBadge,
		steamBadge,
		manifestBadge,
	)

	topSection := container.NewBorder(
		nil, nil,
		nil, container.NewHBox(updateButton, infoButton),
		searchEntry,
	)

	mainContent := container.NewBorder(
		topSection, footer, nil, nil,
		container.NewScroll(content),
	)

	w.SetContent(mainContent)

	var steamApps map[string]steam.App
	var gameRows []*fyne.Container

	go func() {
		time.Sleep(2 * time.Second)
		checkForUpdates(updateButton, messages)

		ticker := time.NewTicker(4 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			checkForUpdates(updateButton, messages)
		}
	}()

	go func() {
		if _, err := r2modman.Find(); err == nil {
			fyne.Do(func() {
				r2modmanBadge.SetText("✅ " + messages.R2ModmanStatus)
			})
		}

		var err error
		steamApps, err = steam.GetApps()
		if err != nil {
			log.Printf("Warning: Could not load Steam apps: %v", err)
			steamApps = make(map[string]steam.App)
		} else {
			fyne.Do(func() {
				steamBadge.SetText(fmt.Sprintf("✅ %s (%d)", messages.SteamStatus, len(steamApps)))
			})
		}

		games, err := profile.FetchGames(cfg.ManifestURL)
		if err != nil {
			log.Printf("Manifest error: %v", err)
			errorIcon := widget.NewIcon(theme.ErrorIcon())
			errorLabel := widget.NewLabel(fmt.Sprintf("%s: %v", messages.Error, err))
			errorContent := container.NewVBox(
				container.NewCenter(errorIcon),
				errorLabel,
			)
			fyne.Do(func() {
				content.Objects = []fyne.CanvasObject{errorContent}
				content.Refresh()
			})
			return
		}

		fyne.Do(func() {
			manifestBadge.SetText(fmt.Sprintf("✅ %s (%d)", messages.ManifestStatus, len(games)))
		})

		if len(games) == 0 {
			warningIcon := widget.NewIcon(theme.WarningIcon())
			noGamesLabel := widget.NewLabel(messages.NoGamesFound)
			noGamesContent := container.NewVBox(
				container.NewCenter(warningIcon),
				noGamesLabel,
			)
			fyne.Do(func() {
				content.Objects = []fyne.CanvasObject{noGamesContent}
				content.Refresh()
			})
			return
		}

		imageCache := make(map[string]*fyne.StaticResource)

		updateGameList := func(filter string) {
			gameList.Objects = nil
			gameRows = nil
			filter = strings.ToLower(filter)

			for _, game := range games {
				if filter != "" && !fuzzyMatch(strings.ToLower(game.Name), filter) {
					continue
				}

				gameRow := createGameRow(game, steamApps, imageCache, messages, cfg, w)
				gameList.Add(gameRow)
				gameRows = append(gameRows, gameRow)
			}
			gameList.Refresh()
		}

		searchEntry.OnChanged = updateGameList

		updateGameList("")

		fyne.Do(func() {
			content.Objects = []fyne.CanvasObject{gameList}
			content.Refresh()
		})

		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				freshGames, err := profile.FetchGames(cfg.ManifestURL)
				if err == nil && len(freshGames) > 0 {
					for i, freshGame := range freshGames {
						if i < len(games) && games[i].Version != freshGame.Version {
							log.Printf("Version change detected for %s: %s -> %s",
								freshGame.Name, games[i].Version, freshGame.Version)
						}
					}
					games = freshGames
					fyne.Do(func() {
						manifestBadge.SetText(fmt.Sprintf("✅ %s (%d)", messages.ManifestStatus, len(games)))
					})
				} else if err != nil {
					log.Printf("Failed to refresh manifest: %v", err)
				}

				fyne.Do(func() {
					updateGameList(searchEntry.Text)
				})
			}
		}()
	}()

	w.ShowAndRun()
}

func showInfoDialog(parent fyne.Window, messages internal.Messages) {
	infoLabel := widget.NewRichTextFromMarkdown(messages.InfoContent)
	infoLabel.Wrapping = fyne.TextWrapWord

	updateCheckButton := widget.NewButtonWithIcon("Nach Updates suchen", theme.ViewRefreshIcon(), func() {
	})

	updateCheckButton.OnTapped = func() {
		go func() {
			fyne.Do(func() {
				updateCheckButton.SetText(messages.CheckingUpdates)
				updateCheckButton.Disable()
			})

			updateInfo, err := updater.CheckForUpdates()

			fyne.Do(func() {
				updateCheckButton.SetText("Nach Updates suchen")
				updateCheckButton.Enable()

				if err != nil {
					dialog.ShowError(fmt.Errorf("Update-Prüfung fehlgeschlagen: %v", err), parent)
					return
				}

				if updateInfo.Available {
					dialog.ShowInformation(messages.UpdateAvailable,
						fmt.Sprintf("Neue Version verfügbar: %s", updateInfo.LatestVersion), parent)
				} else {
					dialog.ShowInformation(messages.NoUpdatesAvailable,
						fmt.Sprintf("Sie verwenden bereits die neueste Version (%s)", updateInfo.CurrentVersion), parent)
				}
			})
		}()
	}

	infoScroll := container.NewScroll(infoLabel)

	windowWidth, windowHeight := GetWindowDimensions()
	dialogWidth := windowWidth - 50
	dialogHeight := windowHeight - 100

	infoScroll.Resize(fyne.NewSize(dialogWidth-50, dialogHeight-150))

	content := container.NewVBox(
		infoScroll,
		widget.NewSeparator(),
		container.NewCenter(updateCheckButton),
	)

	infoDialog := dialog.NewCustom(
		fmt.Sprintf("ℹ️ %s", messages.InfoTitle),
		messages.Close,
		content,
		parent,
	)

	infoDialog.Resize(fyne.NewSize(dialogWidth, dialogHeight))
	infoDialog.Show()
}

func createGameRow(game internal.Game, steamApps map[string]steam.App, imageCache map[string]*fyne.StaticResource, messages internal.Messages, cfg *internal.Config, parent fyne.Window) *fyne.Container {
	headerImg := canvas.NewImageFromResource(nil)
	headerImg.SetMinSize(fyne.NewSize(92, 43))
	headerImg.FillMode = canvas.ImageFillContain

	bg := canvas.NewRectangle(theme.InputBackgroundColor())
	bg.Resize(fyne.NewSize(92, 43))

	imageContainer := container.NewBorder(nil, nil, nil, nil, bg, headerImg)
	imageContainer.Resize(fyne.NewSize(92, 43))

	if game.Header != "" {
		go loadGameIcon(game, headerImg, imageCache)
	}

	nameLabel := widget.NewLabel(game.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel.Wrapping = fyne.TextWrapWord
	nameLabel.Alignment = fyne.TextAlignLeading

	actionBtn := widget.NewButton(messages.LoadingGames, nil)
	actionBtn.Importance = widget.HighImportance

	row := container.NewBorder(
		nil, nil,
		imageContainer,
		actionBtn,
		container.NewPadded(nameLabel),
	)

	var updateRow func()
	updateRow = func() {
		isInstalled := steam.IsGameInstalled(game, steamApps)
		isRunning := steam.IsGameRunning(game)

		profileStatus := profile.GetProfileStatus(game, cfg.TargetDir)

		switch {
		case !isInstalled:
			actionBtn.SetText(messages.NotInstalled)
			actionBtn.SetIcon(theme.WarningIcon())
			actionBtn.Disable()
			return

		case isRunning:
			actionBtn.SetText(messages.StopGame)
			actionBtn.SetIcon(theme.MediaStopIcon())
			actionBtn.Importance = widget.DangerImportance
			actionBtn.Enable()
			actionBtn.OnTapped = func() {
				actionBtn.SetText(messages.Stopping)
				actionBtn.SetIcon(theme.ViewRefreshIcon())
				actionBtn.Importance = widget.MediumImportance
				actionBtn.Disable()

				go func() {
					err := steam.StopGame(game)

					time.Sleep(1 * time.Second)

					fyne.Do(func() {
						if err != nil {
							log.Printf("Failed to stop %s: %v", game.Name, err)
							dialog.ShowError(
								fmt.Errorf("%s: %v", messages.StopFailed, err),
								parent,
							)
						} else {
							log.Printf("Successfully stopped %s", game.Name)
						}
						updateRow()
					})
				}()
			}
			return

		case profileStatus.Installed && profileStatus.HasUpdate:
			actionBtn.SetText(messages.UpdateProfile)
			actionBtn.SetIcon(theme.ViewRefreshIcon())
			actionBtn.Importance = widget.MediumImportance

		case profileStatus.Installed && profileStatus.UpToDate:
			actionBtn.SetText(messages.PlayWithProfile)
			actionBtn.SetIcon(theme.MediaPlayIcon())
			actionBtn.Importance = widget.HighImportance

		case !profileStatus.Installed && game.URL != "":
			actionBtn.SetText(messages.Install)
			actionBtn.SetIcon(theme.DownloadIcon())
			actionBtn.Importance = widget.MediumImportance

		default:
			actionBtn.SetText(messages.PlayGame)
			actionBtn.SetIcon(theme.MediaPlayIcon())
			actionBtn.Importance = widget.HighImportance
		}

		actionBtn.Enable()

		currentProfileStatus := profile.GetProfileStatus(game, cfg.TargetDir)

		if !currentProfileStatus.Installed && game.URL != "" {
			actionBtn.OnTapped = func() {
				actionBtn.SetText(messages.Installing)
				actionBtn.SetIcon(theme.ViewRefreshIcon())
				actionBtn.Importance = widget.MediumImportance
				actionBtn.Disable()

				go func() {
					err := profile.DownloadAndInstall(game, cfg.TargetDir)

					fyne.Do(func() {
						if err != nil {
							log.Printf("Failed to install profile for %s: %v", game.Name, err)
							dialog.ShowError(
								fmt.Errorf("%s: %v", messages.InstallationFailed, err),
								parent,
							)
						} else {
							log.Printf("Successfully installed profile for %s", game.Name)
						}
						updateRow()
					})
				}()
			}
		} else if currentProfileStatus.Installed && currentProfileStatus.HasUpdate && game.URL != "" {
			actionBtn.OnTapped = func() {
				actionBtn.SetText(messages.Updating)
				actionBtn.SetIcon(theme.ViewRefreshIcon())
				actionBtn.Importance = widget.MediumImportance
				actionBtn.Disable()

				go func() {
					err := profile.DeleteProfile(game)
					if err != nil {
						log.Printf("Warning: Failed to delete old profile for %s: %v", game.Name, err)
					}

					err = profile.DownloadAndInstall(game, cfg.TargetDir)

					fyne.Do(func() {
						if err != nil {
							log.Printf("Failed to update profile for %s: %v", game.Name, err)
							dialog.ShowError(
								fmt.Errorf("%s: %v", messages.UpdateFailed, err),
								parent,
							)
						} else {
							log.Printf("Successfully updated profile for %s", game.Name)
						}
						updateRow()
					})
				}()
			}
		} else {
			actionBtn.OnTapped = func() {
				go func() {
					err := steam.LaunchGame(game, cfg.TargetDir, steamApps)
					fyne.Do(func() {
						if err != nil {
							log.Printf("Failed to launch %s: %v", game.Name, err)
							dialog.ShowError(
								fmt.Errorf("%s: %v", messages.LaunchFailed, err),
								parent,
							)
						}
					})
				}()
			}
		}
	}

	updateRow()

	rowWithSeparator := container.NewVBox(
		container.NewPadded(row),
		widget.NewSeparator(),
	)

	return rowWithSeparator
}

func loadGameIcon(game internal.Game, headerImg *canvas.Image, imageCache map[string]*fyne.StaticResource) {
	if cached, exists := imageCache[game.Name]; exists {
		headerImg.Resource = cached
		headerImg.Refresh()
		return
	}

	go func() {
		log.Printf("Loading image for %s: %s", game.Name, game.Header)
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(game.Header)
		if err != nil {
			log.Printf("Failed to fetch image for %s: %v", game.Name, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Bad status for image %s: %d", game.Name, resp.StatusCode)
			return
		}

		imgData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read image data for %s: %v", game.Name, err)
			return
		}

		log.Printf("Loaded %d bytes for image %s", len(imgData), game.Name)

		resource := fyne.NewStaticResource(game.Name+"_header", imgData)

		imageCache[game.Name] = resource

		fyne.Do(func() {
			headerImg.Resource = resource
			headerImg.Refresh()
			log.Printf("Updated image for %s", game.Name)
		})
	}()
}

func checkForUpdates(updateButton *widget.Button, messages internal.Messages) {
	go func() {
		updateInfo, err := updater.CheckForUpdates()
		if err != nil {
			log.Printf("Failed to check for updates: %v", err)
			return
		}

		if updateInfo.Available {
			fyne.Do(func() {
				updateButton.SetText(messages.UpdateAvailable)
				updateButton.Show()
			})
		}
	}()
}

func showUpdateDialog(parent fyne.Window, messages internal.Messages) {
	updateInfo, err := updater.CheckForUpdates()
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}

	if !updateInfo.Available {
		dialog.ShowInformation(messages.NoUpdatesAvailable,
			fmt.Sprintf("Sie verwenden bereits die neueste Version (%s)", updateInfo.CurrentVersion), parent)
		return
	}

	content := fmt.Sprintf(`**Neue Version verfügbar!**

**Aktuelle Version:** %s
**Neue Version:** %s

**Änderungen:**
%s

Möchten Sie jetzt aktualisieren?`,
		updateInfo.CurrentVersion,
		updateInfo.LatestVersion,
		updateInfo.ReleaseNotes)

	updateLabel := widget.NewRichTextFromMarkdown(content)
	updateLabel.Wrapping = fyne.TextWrapWord

	updateScroll := container.NewScroll(updateLabel)
	updateScroll.Resize(fyne.NewSize(500, 300))

	updateDialog := dialog.NewCustomConfirm(
		messages.UpdateDialogTitle,
		messages.UpdateNow,
		messages.UpdateLater,
		updateScroll,
		func(update bool) {
			if update {
				performUpdate(parent, messages, updateInfo)
			}
		},
		parent,
	)

	updateDialog.Resize(fyne.NewSize(550, 400))
	updateDialog.Show()
}

func performUpdate(parent fyne.Window, messages internal.Messages, updateInfo *updater.UpdateInfo) {
	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel(messages.UpdateDownloading)

	progressContent := container.NewVBox(
		progressLabel,
		progressBar,
	)

	progressDialog := dialog.NewCustomWithoutButtons(
		messages.UpdateDialogTitle,
		progressContent,
		parent,
	)
	progressDialog.Show()

	go func() {
		tempFile, err := updater.DownloadUpdate(updateInfo.DownloadURL, func(downloaded, total int64) {
			if total > 0 {
				progress := float64(downloaded) / float64(total)
				fyne.Do(func() {
					progressBar.SetValue(progress)
					progressLabel.SetText(fmt.Sprintf("%s (%d%%)",
						messages.UpdateDownloading, int(progress*100)))
				})
			}
		})

		if err != nil {
			fyne.Do(func() {
				progressDialog.Hide()
				dialog.ShowError(fmt.Errorf("%s: %v", messages.UpdateError, err), parent)
			})
			return
		}

		fyne.Do(func() {
			progressBar.SetValue(1.0)
			progressLabel.SetText(messages.UpdateInstalling)
		})

		if err := updater.ApplyUpdate(tempFile); err != nil {
			fyne.Do(func() {
				progressDialog.Hide()
				dialog.ShowError(fmt.Errorf("%s: %v", messages.UpdateError, err), parent)
			})
		}
	}()
}
