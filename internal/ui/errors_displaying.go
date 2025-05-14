package ui

import (
	"errors"
	"vocabulary/internal/app"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Returns a dialog styled similary to standard fyne one, constructed with dialog.NewError(...).
func dialogOfTextErr(errText string, window fyne.Window) dialog.Dialog {
	lb := widget.NewLabel(errText)

	lb.Alignment = fyne.TextAlignCenter

	dlg := dialog.NewCustom(lang.L("Error"), lang.L("OK"), lb, window)

	dlg.SetIcon(theme.ErrorIcon())

	return dlg
}

func identifyTranslateErr(err error) string {
	if errors.Is(err, app.ErrNotEnoughPhrasesInLesson) {
		return lang.L("Not enough phrases in lesson")
	}

	if errors.Is(err, app.ErrTaskAlreadyTaken) {
		return lang.L("Task pre-loading error")
	}

	return ""
}

func showErr(err error, window fyne.Window) {
	var (
		translatedErrText = identifyTranslateErr(err)
		dlg               dialog.Dialog
	)

	if translatedErrText == "" {
		dlg = dialog.NewError(err, window)
	} else {
		dlg = dialogOfTextErr(translatedErrText, window)
	}

	dlg.Show()
}
