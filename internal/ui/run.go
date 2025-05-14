package ui

import (
	"context"
	"embed"
	"log"
	"sync"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
)

//go:embed translation
var translations embed.FS

func Run(app Application) {
	fyneApp := fyneapp.NewWithID("unique")

	err := lang.AddTranslationsFS(translations, "translation")

	if err != nil {
		log.Fatal(err)
	}

	wg := &sync.WaitGroup{}

	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	mainWindow := fyneApp.NewWindow("vocabulary")

	mainWindow.Resize(fyne.NewSize(1000, 500))

	openMainMenu(ctx, wg, mainWindow, app)

	mainWindow.ShowAndRun()
}
