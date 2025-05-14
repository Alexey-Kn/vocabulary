package advanced

import (
	"context"
	"vocabulary/internal/app"
)

type oneOptionChoiceTask struct {
	PhraseToTranslate string
	AvailableOptions  []string
	RightAnswer       int
	IsInverted        bool
	PhraseIndex       int
	Solved            func(app.PhraseLearningTask, bool)

	alreadyAnswered bool
}

var _ app.ChooseRightOption = (*oneOptionChoiceTask)(nil)

func (t *oneOptionChoiceTask) Phrase() string {
	return t.PhraseToTranslate
}

func (t *oneOptionChoiceTask) Inverted() bool {
	return t.IsInverted
}

func (t *oneOptionChoiceTask) Options() []string {
	return t.AvailableOptions
}

func (t *oneOptionChoiceTask) Right(_ context.Context, option int) (bool, error) {
	answerIsCorrect := option == t.RightAnswer

	if !t.alreadyAnswered {
		t.alreadyAnswered = true

		t.Solved(t, answerIsCorrect)
	}

	return answerIsCorrect, nil
}

func (t *oneOptionChoiceTask) GetRightAnswer(context.Context) (int, error) {
	if !t.alreadyAnswered {
		t.alreadyAnswered = true

		t.Solved(t, false)
	}

	return t.RightAnswer, nil
}
