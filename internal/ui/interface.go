package ui

import (
	"vocabulary/internal/app"
)

type Application interface {
	OpenFile(path string) bool
	FilePath() string

	SetLessonMode(app.LessonMode)
	GetLessonMode() app.LessonMode

	AvailableTopics() []string
	ChooseTopic(string)
	Topic() string

	ProgressRecoveryIsAvailable() bool
	BeginLesson(recoverProgress bool) (app.Lesson, error)
}
