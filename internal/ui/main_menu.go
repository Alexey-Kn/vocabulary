package ui

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type mainMenu struct {
	menu

	filePathEntry  *widget.Entry
	learnButton    *widget.Button
	topicSelection *widget.Select
}

func (m *mainMenu) topicChanged(topic string) {
	m.app.ChooseTopic(topic)

	m.update()
}

func (m *mainMenu) filePathChanged(newPath string) {
	m.app.OpenFile(newPath)

	m.update()
}

func (m *mainMenu) chooseFileButtonPressed() {
	dlg := dialog.NewFileOpen(m.fileDialogClosed, m.mainWindow)

	dlg.SetView(dialog.ListView)
	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx"}))
	dlg.Resize(m.mainWindow.Canvas().Size())

	dlg.Show()
}

func (m *mainMenu) fileDialogClosed(reader fyne.URIReadCloser, err error) {
	if reader == nil || err != nil {
		return
	}

	path := reader.URI().Path()

	reader.Close()

	m.app.OpenFile(path)

	m.update()
}

func (m *mainMenu) showError(err error) {
	showErr(err, m.mainWindow)
}

func (m *mainMenu) learnButtonPressed() {
	if m.app.ProgressRecoveryIsAvailable() {
		dialogTextLabel := widget.NewLabel(lang.L("Recover progress?"))

		dialogTextLabel.Alignment = fyne.TextAlignCenter

		dlg := dialog.NewCustom(lang.L("Progress recovery"), "OK", dialogTextLabel, m.mainWindow)

		dlg.SetButtons(
			[]fyne.CanvasObject{
				widget.NewButton(
					lang.L("Yes"),
					func() {
						dlg.Dismiss()

						m.beginLesson(true)
					},
				),
				widget.NewButton(
					lang.L("No"),
					func() {
						dlg.Dismiss()

						m.beginLesson(false)
					},
				),
			},
		)

		dlg.Show()
	} else {
		m.beginLesson(false)
	}
}

func (m *mainMenu) beginLesson(recoverProgress bool) {
	lesson, err := m.app.BeginLesson(recoverProgress)

	if err != nil {
		m.showError(err)

		return
	}

	openLesson(m.ctx, m.wg, m.mainWindow, lesson, m.app)
}

func (m *mainMenu) update() {
	m.learnButton.OnTapped = nil
	m.topicSelection.OnChanged = nil
	m.filePathEntry.OnChanged = nil

	path := m.app.FilePath()

	if path != "" {
		m.filePathEntry.SetText(path)
	}

	topics := m.app.AvailableTopics()

	m.topicSelection.SetOptions(topics) //can call topicChanged

	if len(topics) > 0 {
		m.topicSelection.Enable()
		m.topicSelection.SetSelected(m.app.Topic())
	} else {
		m.topicSelection.ClearSelected()
		m.topicSelection.Disable()
	}

	if m.topicSelection.SelectedIndex() >= 0 {
		m.learnButton.Enable()
	} else {
		m.learnButton.Disable()
	}

	m.learnButton.OnTapped = m.learnButtonPressed
	m.topicSelection.OnChanged = m.topicChanged
	m.filePathEntry.OnChanged = m.filePathChanged
}

// Opens a menu for choice an excel file and its' sheet.
func openMainMenu(ctx context.Context, wg *sync.WaitGroup, mainWindow fyne.Window, app Application) {
	menu := &mainMenu{
		menu: menu{
			app:        app,
			mainWindow: mainWindow,
			ctx:        ctx,
			wg:         wg,
		},
		filePathEntry:  widget.NewEntry(),
		learnButton:    widget.NewButton(lang.L("Begin lesson"), nil),
		topicSelection: widget.NewSelect([]string{}, nil),
	}

	menu.learnButton.Importance = widget.HighImportance

	menu.learnButton.OnTapped = menu.learnButtonPressed
	menu.topicSelection.OnChanged = menu.topicChanged
	menu.filePathEntry.OnChanged = menu.filePathChanged

	menu.update()

	instructionsLabel := widget.NewLabel(
		lang.X("excel_file_choice_instructions", "Choose the Excel file contains phrases to learn. Each sheet should have two columns: the phrase and its translation without any header in the first row."),
	)

	instructionsLabel.Wrapping = fyne.TextWrapWord

	mainWindow.SetContent(
		container.NewBorder(
			container.NewHBox(
				menu.learnButton,
				layout.NewSpacer(),
			),
			nil,
			nil,
			nil,
			container.NewVBox(
				layout.NewSpacer(),
				instructionsLabel,
				layout.NewSpacer(),
				container.New(
					layout.NewFormLayout(),
					widget.NewLabel(
						lang.L("File")+":",
					),
					container.NewBorder(
						nil,
						nil,
						nil,
						widget.NewButton("...", menu.chooseFileButtonPressed),
						menu.filePathEntry,
					),
					widget.NewLabel(
						lang.L("Topic")+":",
					),
					menu.topicSelection,
				),
				layout.NewSpacer(),
			),
		),
	)
}
