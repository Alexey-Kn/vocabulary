package advanced

import (
	"context"
	"strings"
	"vocabulary/internal/app"
)

type tranclateManuallyTask struct {
	PhraseToTranslate app.PhraseWithTranslation
	IsInverted        bool
	PhraseIndex       int
	Solved            func(app.PhraseLearningTask, bool)

	alreadyAnswered bool
}

var _ app.TranslateManually = (*tranclateManuallyTask)(nil)

func (t *tranclateManuallyTask) GetRightAnswer(context.Context) (string, error) {
	if !t.alreadyAnswered {
		t.alreadyAnswered = true

		t.Solved(t, false)
	}

	return t.PhraseToTranslate.Translation, nil
}

func (t *tranclateManuallyTask) Inverted() bool {
	return t.IsInverted
}

func (t *tranclateManuallyTask) Phrase() string {
	return t.PhraseToTranslate.Phrase
}

func (t *tranclateManuallyTask) Right(_ context.Context, translation string) (bool, error) {
	toCompareWith := t.PhraseToTranslate.Translation

	for replaceWhat, replaceFor := range replacement {
		translation = strings.ReplaceAll(translation, string(replaceWhat), string(replaceFor))
		toCompareWith = strings.ReplaceAll(toCompareWith, string(replaceWhat), string(replaceFor))
	}

	translation = strings.TrimSpace(translation)
	toCompareWith = strings.TrimSpace(toCompareWith)

	answerIsCorrect := strings.EqualFold(translation, toCompareWith)

	if !t.alreadyAnswered {
		t.alreadyAnswered = true

		t.Solved(t, answerIsCorrect)
	}

	return answerIsCorrect, nil
}
