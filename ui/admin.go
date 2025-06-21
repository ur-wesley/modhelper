package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ur-wesley/modhelper/internal"
	"github.com/ur-wesley/modhelper/internal/config"
)

func RunAdmin() {
	a := app.NewWithID("com.urwesley.modhelper.admin")

	messages := internal.German()

	w := a.NewWindow(internal.AppName + " - " + messages.AdminMode)
	w.Resize(fyne.NewSize(600, 400))

	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		cfg = &internal.Config{
			ManifestURL: "https://gist.githubusercontent.com/ur-wesley/8e93a37dc70b7d8161e94fc62df061ee/raw",
			TargetDir:   config.GetDefaultProfileDir(),
		}
	}

	manifestEntry := widget.NewEntry()
	manifestEntry.SetText(cfg.ManifestURL)
	manifestEntry.MultiLine = false

	targetDirEntry := widget.NewEntry()
	targetDirEntry.SetText(cfg.TargetDir)
	targetDirEntry.MultiLine = false

	form := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:   messages.ManifestURL,
				Widget: container.NewBorder(nil, nil, widget.NewIcon(theme.DocumentIcon()), nil, manifestEntry),
			},
			{
				Text:   messages.TargetDir,
				Widget: container.NewBorder(nil, nil, widget.NewIcon(theme.FolderIcon()), nil, targetDirEntry),
			},
		},
	}

	saveBtn := widget.NewButtonWithIcon(messages.Save, theme.DocumentSaveIcon(), func() {
		newCfg := &internal.Config{
			ManifestURL: manifestEntry.Text,
			TargetDir:   targetDirEntry.Text,
		}

		err := config.Save(newCfg)
		if err != nil {
			log.Printf("Failed to save config: %v", err)
			errorDialog := dialog.NewError(err, w)
			errorDialog.Show()
			return
		}

		log.Println("Configuration saved successfully")
		successDialog := dialog.NewInformation(
			"✅ Konfiguration gespeichert",
			"Die Einstellungen wurden erfolgreich gespeichert.",
			w,
		)
		successDialog.Show()
	})
	saveBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButtonWithIcon(messages.Cancel, theme.CancelIcon(), func() {
		w.Close()
	})

	buttons := container.NewHBox(
		saveBtn,
		cancelBtn,
	)

	infoIcon := widget.NewIcon(theme.InfoIcon())
	infoText := widget.NewRichTextFromMarkdown(`**Konfiguration**

• **Manifest-URL**: URL zum JSON-Manifest mit Spiellisten
• **Zielordner**: Pfad für r2modman Profile Installation

Änderungen werden sofort nach dem Speichern aktiv.`)
	infoText.Wrapping = fyne.TextWrapWord

	infoContainer := container.NewBorder(
		nil, nil,
		infoIcon, nil,
		infoText,
	)

	content := container.NewVBox(
		container.NewPadded(infoContainer),
		widget.NewSeparator(),
		form,
		widget.NewSeparator(),
		container.NewCenter(buttons),
	)

	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}
