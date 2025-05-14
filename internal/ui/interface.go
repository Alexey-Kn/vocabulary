package ui

import (
	"vocabulary/internal/app"
)

type Application interface {
	OpenFile(path string) bool
	FilePath() string

	AvailableTopics() []string
	ChooseTopic(string)
	Topic() string

	ProgressRecoveryIsAvailable() bool
	BeginLesson(recoverProgress bool) (app.Lesson, error)
}
