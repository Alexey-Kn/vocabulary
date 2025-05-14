package app

import "errors"

var (
	ErrNotEnoughPhrasesInLesson = errors.New("not enough phrases in lesson")

	ErrTaskAlreadyTaken = errors.New("task has already been taken")
)
