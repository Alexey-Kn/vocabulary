package app

import "context"

type Lesson interface {
	Next(ctx context.Context) (PhraseLearningTask, error)
}

type PhraseLearningTask interface {
	Phrase() string
	Inverted() bool
}

type ChooseRightOption interface {
	PhraseLearningTask
	Options() []string
	Right(context.Context, int) (bool, error)
	GetRightAnswer(context.Context) (int, error)
}

type TranslateManually interface {
	PhraseLearningTask
	Right(context.Context, string) (bool, error)
	GetRightAnswer(context.Context) (string, error)
}
