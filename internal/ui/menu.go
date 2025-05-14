package ui

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
)

type menu struct {
	ctx        context.Context
	app        Application
	mainWindow fyne.Window
	wg         *sync.WaitGroup
}

// Calls Async func in a new goroutine and inUIGoroutine as fyne.Do() argument.
// inUIGoroutine func will be called only if context haven't been cancelled.
// Uses wg to control exiting of new goroutine.
func (m *menu) Async(async func(context.Context), inUIGouroutine func()) {
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()

		async(m.ctx)

		if m.ctx.Err() == nil {
			fyne.Do(inUIGouroutine)
		}
	}()
}
