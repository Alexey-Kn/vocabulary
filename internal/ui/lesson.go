package ui

import (
	"context"
	"sync"
	"time"
	"vocabulary/internal/app"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type enableAble interface {
	Enable()
	Disable()
}

func setEnabled(widget enableAble, flag bool) {
	if flag {
		widget.Enable()
	} else {
		widget.Disable()
	}
}

type lessonMenu struct {
	menu

	toMainMenu, showRightAnswer *widget.Button

	translation      *widget.Entry
	checkTranslation *widget.Button

	phraseToTranslate  *widget.Label
	translationOptions []*widget.Button

	lesson app.Lesson
	task   app.PhraseLearningTask

	// This flag needed to skip pause for right answer displaying
	// when user tapped "Show right answer" button.
	skipPauseBeforeDisplayingNextTask bool
	translateManuallyRightAnswer      string

	// Used by ignoreUserActions(flag bool) and ignoringUserActions().
	ignoringUserActionsFlag byte

	// Calls setWidgetsEnabled(false) afer expiration TIME_BEFORE_SHOWING_WAITING_SCREEN
	// since Enter() call and setWidgetsEnabled(true) after expiration of MIN_TIME_OF_WAITING_SCREEN_DISPLAYING
	// since setWidgetsEnabled(false) call (or later when Leave() was called later).
	slowOperationsIndication *indicatedSlowOperation
}

func (m *lessonMenu) showError(err error) {
	if m.ctx.Err() != nil {
		return
	}

	showErr(err, m.mainWindow)
}

func (m *lessonMenu) phraseTranslatedManually() {
	if m.ignoringUserActions() {
		return
	}

	if m.translation.Text == "" && m.translation.PlaceHolder != "" {
		lb := widget.NewLabel(lang.L("Input translation. The notice in the input field is a background suggestion."))

		lb.Alignment = fyne.TextAlignCenter

		dialog.NewCustom(
			lang.L("Input translation"),
			lang.L("OK"),
			lb,
			m.mainWindow,
		).Show()
	}

	var (
		t       = m.task.(app.TranslateManually)
		isRight bool
		err     error
	)

	m.async(
		func(ctx context.Context) {
			isRight, err = t.Right(m.ctx, m.translation.Text)
		},
		func() {
			if err != nil {
				m.showError(err)

				return
			}

			var newImportance widget.Importance

			if isRight {
				newImportance = widget.SuccessImportance

				pause := TIME_TO_DEMONSTRATE_RIGHT_ANSWER

				if m.skipPauseBeforeDisplayingNextTask {
					pause = 0
				}

				m.next(pause)
			} else {
				newImportance = widget.DangerImportance
			}

			m.checkTranslation.Importance = newImportance
			m.checkTranslation.Refresh()
		},
	)
}

func (m *lessonMenu) translationOptionChosen(option int) {
	if m.ignoringUserActions() {
		return
	}

	var (
		t       = m.task.(app.ChooseRightOption)
		isRight bool
		err     error
	)

	m.async(
		func(ctx context.Context) {
			isRight, err = t.Right(m.ctx, option)
		},
		func() {
			if err != nil {
				m.showError(err)

				return
			}

			var newImportance widget.Importance

			if isRight {
				newImportance = widget.SuccessImportance

				pause := TIME_TO_DEMONSTRATE_RIGHT_ANSWER

				if m.skipPauseBeforeDisplayingNextTask {
					pause = 0
				}

				m.next(pause)
			} else {
				newImportance = widget.DangerImportance
			}

			buttonOfAnswer := m.translationOptions[option]

			buttonOfAnswer.Importance = newImportance
			buttonOfAnswer.Refresh()
		},
	)
}

func (m *lessonMenu) next(delay time.Duration) {
	var (
		newTask app.PhraseLearningTask
		err     error
	)

	m.async(
		func(ctx context.Context) {
			momentToWaitFor := time.Now().Add(delay)

			newTask, err = m.lesson.Next(ctx)

			if delay > 0 {
				wait := time.Until(momentToWaitFor)

				if wait > 0 {
					select {
					case <-ctx.Done():
					case <-time.After(wait):
					}
				}
			}
		},
		func() {
			if err != nil {
				m.showError(err)

				return
			}

			m.showTask(newTask)
		},
	)
}

// The same as menu.Async(), but calls m.slowOperationsIndication.Enter()/Leave()
// and m.ignoreUserActions(true/false) do display operation status.
func (m *lessonMenu) async(async func(context.Context), inUIGouroutine func()) {
	m.slowOperationsIndication.Enter()
	m.ignoreUserActions(true)

	m.menu.Async(
		func(ctx context.Context) {
			defer m.slowOperationsIndication.LeaveAndWait()

			async(ctx)
		},
		func() {
			defer m.ignoreUserActions(false)

			inUIGouroutine()
		},
	)
}

func (m *lessonMenu) showTask(newTask app.PhraseLearningTask) {
	var (
		content fyne.CanvasObject
		focus   fyne.Focusable
	)

	m.task = newTask

	m.phraseToTranslate.SetText(m.task.Phrase())

	m.skipPauseBeforeDisplayingNextTask = false

	switch t := m.task.(type) {
	case app.TranslateManually:
		content = container.NewVBox(
			layout.NewSpacer(),
			container.NewCenter(
				m.phraseToTranslate,
			),
			layout.NewSpacer(),
			container.NewBorder(
				nil,
				nil,
				widget.NewLabel(lang.X("tasks.translation", "Translation")+":"), //key "Translation" conflicts with something
				m.checkTranslation,
				m.translation,
			),
			layout.NewSpacer(),
		)

		m.checkTranslation.Importance = widget.MediumImportance

		m.translation.SetText("")
		m.translation.SetPlaceHolder("")

		m.translateManuallyRightAnswer = ""

		focus = m.translation
	case app.ChooseRightOption:
		var (
			options      = t.Options()
			optionsCO    = make([]fyne.CanvasObject, len(options))
			optionButton *widget.Button
		)

		if len(options) < len(m.translationOptions) {
			m.translationOptions = m.translationOptions[:len(options)]
		}

		for i, option := range options {
			if i >= len(m.translationOptions) {
				iCopy := i

				optionButton = widget.NewButton(
					option,
					func() {
						m.translationOptionChosen(iCopy)
					},
				)

				m.translationOptions = append(m.translationOptions, optionButton)
			} else {
				optionButton = m.translationOptions[i]

				optionButton.SetText(option)

				optionButton.Importance = widget.MediumImportance
			}

			optionsCO[i] = optionButton
		}

		content = container.NewVBox(
			layout.NewSpacer(),
			container.NewCenter(
				m.phraseToTranslate,
			),
			layout.NewSpacer(),
			container.NewHBox(
				layout.NewSpacer(),
				container.NewGridWithColumns(4, optionsCO...),
				layout.NewSpacer(),
			),
			layout.NewSpacer(),
		)
	}

	m.mainWindow.SetContent(
		container.NewBorder(
			container.NewHBox(
				m.toMainMenu,
				m.showRightAnswer,
				layout.NewSpacer(),
			),
			nil,
			nil,
			nil,
			content,
		),
	)

	if focus != nil {
		m.mainWindow.Canvas().Focus(focus)
	}
}

// Needed to avoid responding to double-click on buttons and other
// user' actions while the program is loading. Can be called recursively.
func (m *lessonMenu) ignoreUserActions(flag bool) {
	if flag {
		m.ignoringUserActionsFlag++
	} else {
		m.ignoringUserActionsFlag--
	}
}

func (m *lessonMenu) ignoringUserActions() bool {
	return m.ignoringUserActionsFlag > 0
}

// Used to show "loading" status. Doesn't disable "To main menu" button, so
// user can cancel any operation without exiting the application.
func (m *lessonMenu) setWidgetsEnabled(flag bool) {
	setEnabled(m.showRightAnswer, flag)
	setEnabled(m.translation, flag)
	setEnabled(m.checkTranslation, flag)

	for _, button := range m.translationOptions {
		setEnabled(button, flag)
	}
}

func (m *lessonMenu) showRightAnswerButtonTapped() {
	if m.ignoringUserActions() {
		return
	}

	switch t := m.task.(type) {
	case app.TranslateManually:
		if m.translateManuallyRightAnswer == "" {
			var (
				rightTranslation string
				err              error
			)

			m.async(
				func(ctx context.Context) {
					rightTranslation, err = t.GetRightAnswer(m.ctx)
				},
				func() {
					if err != nil {
						m.showError(err)

						return
					}

					m.translation.SetText("")
					m.translation.SetPlaceHolder(string([]rune(rightTranslation)[:1]) + "...")
					m.translateManuallyRightAnswer = rightTranslation
				},
			)
		} else {
			m.translation.SetText("")
			m.translation.SetPlaceHolder(m.translateManuallyRightAnswer)
		}

	case app.ChooseRightOption:
		var (
			rightoptionIndex int
			err              error
		)

		m.async(
			func(ctx context.Context) {
				rightoptionIndex, err = t.GetRightAnswer(m.ctx)
			},
			func() {
				if err != nil {
					m.showError(err)

					return
				}

				m.skipPauseBeforeDisplayingNextTask = true

				rightOptionButton := m.translationOptions[rightoptionIndex]

				rightOptionButton.Importance = widget.SuccessImportance
				rightOptionButton.Refresh()
			},
		)
	}
}

// Opens UI form of the lesson. It is expected that object of the lesson recieved in parameter
// can call remote resources and respond very slowly. The UI form will handle waiting of these slow
// operations properly (user will see inactive widgets for cases of waiting during more than
// TIME_TO_DEMONSTRATE_RIGHT_ANSWER period).
func openLesson(ctx context.Context, wg *sync.WaitGroup, mainWindow fyne.Window, lesson app.Lesson, app Application) error {
	menuContext, cancelMenuContext := context.WithCancel(ctx)

	m := &lessonMenu{
		menu: menu{
			ctx:        menuContext,
			app:        app,
			mainWindow: mainWindow,
			wg:         wg,
		},
		lesson:             lesson,
		toMainMenu:         widget.NewButton(lang.L("To main menu"), nil),
		showRightAnswer:    widget.NewButton(lang.L("Show right answer"), nil),
		translation:        widget.NewEntry(),
		checkTranslation:   widget.NewButton(lang.L("Check"), nil),
		phraseToTranslate:  widget.NewLabel(""),
		translationOptions: make([]*widget.Button, 0, 8),
		task:               nil,
	}

	m.slowOperationsIndication = newSlowOperation(
		menuContext,
		wg,
		TIME_BEFORE_SHOWING_WAITING_SCREEN,
		MIN_TIME_OF_WAITING_SCREEN_DISPLAYING,
		func() {
			fyne.DoAndWait(
				func() {
					m.setWidgetsEnabled(false)
				},
			)
		},
		func() {
			fyne.DoAndWait(
				func() {
					m.setWidgetsEnabled(true)
				},
			)
		},
	)

	m.toMainMenu.Importance = widget.HighImportance

	m.showRightAnswer.Importance = widget.HighImportance

	m.translation.OnSubmitted = func(s string) {
		m.phraseTranslatedManually()
	}

	m.showRightAnswer.OnTapped = m.showRightAnswerButtonTapped

	m.checkTranslation.OnTapped = m.phraseTranslatedManually

	m.translation.OnChanged = func(s string) {
		m.checkTranslation.Importance = widget.MediumImportance
		m.checkTranslation.Refresh()
	}

	m.toMainMenu.OnTapped = func() {
		cancelMenuContext()

		openMainMenu(ctx, m.wg, m.mainWindow, m.app)
	}

	m.next(0)

	return nil
}
