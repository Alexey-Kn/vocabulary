package advanced

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	mathrand "math/rand"
	"vocabulary/internal/app"
	"vocabulary/internal/random"
)

type kindOfTask int

const (
	kindOfTaskChooseOneOption kindOfTask = iota
	kindOfTaskTranslateManually
)

type PhraseLearningStatistics struct {
	CountGuessedOOS         uint32
	CountFailedOOS          uint32
	CountAnsweredTM         uint32
	CountFailedTM           uint32
	CountGuessedOOSInverted uint32
	CountFailedOOSInverted  uint32
	CountAnsweredTMInverted uint32
	CountFailedTMInverted   uint32
}

func (s *PhraseLearningStatistics) IsEmpty() bool {
	return s.CountGuessedOOS == 0 &&
		s.CountFailedOOS == 0 &&
		s.CountAnsweredTM == 0 &&
		s.CountFailedTM == 0 &&
		s.CountGuessedOOSInverted == 0 &&
		s.CountFailedOOSInverted == 0 &&
		s.CountAnsweredTMInverted == 0 &&
		s.CountFailedTMInverted == 0
}

type PhraseWithLearningStatistics struct {
	Phrase             app.PhraseWithTranslation
	LearningStatistics PhraseLearningStatistics
}

type phraseWithStatisticsAndTasksIndexes struct {
	Phrase             app.PhraseWithTranslation
	LearningStatistics PhraseLearningStatistics

	IndexOfChooseRightOptionTask         int
	IndexOfTranslateManuallyTask         int
	IndexOfChooseRightOptionInvertedTask int
	IndexOfTranslateManuallyInvertedTask int
}

type taskCreationData struct {
	PhraseIndex int
	Inverted    bool
	KnidOfTask  kindOfTask
}

type weightWithIndex struct {
	Index  int
	Weight float64
}

type phraseWithTasksWeights struct {
	PhraseIndex int
	Tasks       []weightWithIndex
}

// The implementation of app.Lesson; provides a
// specific order of tasks based on answers statistics.
// Tasks order is random, but probability of choice of concrete
// task depends on user answers.
//
// Mehod  calculateWeightOfTask() contains the algorithm of prioritizing tasks.
type Lesson struct {
	//Slace phrases is four times shorter than tasksProperties because each phrase is connected
	//with 4 tasks: choose one option, choose one option inverted, translate manually and
	//translate manually inverted.

	phrases         []phraseWithStatisticsAndTasksIndexes
	tasksProperties []taskCreationData

	//Each value recieved from tasksSelector.Get() method is an index in tasksProperties slice.
	//Weight of i-th value provides a probability of return i-th task by Next() method.
	//(weight - unnormilized probability).
	tasksSelector *random.DiscreteRandomVariable
	randSource    *mathrand.Rand

	//Short history of used phrases. Filled from last to first element. Can contain nil at the beginning
	//of lesson. Number 4 can be increased to decrease the probability of too often usage of one phrase.
	//
	//Method updateLastPhrasesWeights() changes weights of tasks connected with these phrases by such a alogick:
	//	* last element:  0 (just used this phrase)
	//	* i-th element:  original weight / (i + 1)
	//	* ...
	//	* first element: original weight
	lastPhrasesToNotRepeat [4]*phraseWithTasksWeights
}

var _ app.Lesson = (*Lesson)(nil)

func New(phrases []app.PhraseWithTranslation) (*Lesson, error) {
	withStatistics := make([]PhraseWithLearningStatistics, len(phrases))

	for i, phrase := range phrases {
		withStatsLine := &withStatistics[i]

		withStatsLine.Phrase = phrase
		withStatsLine.LearningStatistics = PhraseLearningStatistics{}
	}

	return NewWithProgress(withStatistics)
}

func NewWithProgress(phrases []PhraseWithLearningStatistics) (*Lesson, error) {
	var (
		phrasesWithStatistics = make([]phraseWithStatisticsAndTasksIndexes, len(phrases))
		tasksProperties       = make([]taskCreationData, 0, len(phrases)*4)
		weights               = make([]float64, 0, len(phrases)*4)
	)

	addTask := func(i int, stats *PhraseLearningStatistics, kindOfTask kindOfTask, inverted bool) int {
		tcd := taskCreationData{
			PhraseIndex: i,
			Inverted:    inverted,
			KnidOfTask:  kindOfTask,
		}

		index := len(tasksProperties)

		tasksProperties = append(
			tasksProperties,
			tcd,
		)

		weights = append(
			weights,
			calculateWeightOfTask(
				stats,
				tcd.KnidOfTask,
				tcd.Inverted,
			),
		)

		return index
	}

	for i, phrase := range phrases {
		pwsati := phraseWithStatisticsAndTasksIndexes{
			Phrase:             phrase.Phrase,
			LearningStatistics: phrase.LearningStatistics,
		}

		pwsati.IndexOfChooseRightOptionTask = addTask(i, &pwsati.LearningStatistics, kindOfTaskChooseOneOption, false)

		pwsati.IndexOfChooseRightOptionInvertedTask = addTask(i, &pwsati.LearningStatistics, kindOfTaskChooseOneOption, true)

		pwsati.IndexOfTranslateManuallyTask = addTask(i, &pwsati.LearningStatistics, kindOfTaskTranslateManually, false)

		pwsati.IndexOfTranslateManuallyInvertedTask = addTask(i, &pwsati.LearningStatistics, kindOfTaskTranslateManually, true)

		phrasesWithStatistics[i] = pwsati
	}

	randomKey := make([]byte, 8)

	cryptorand.Read(randomKey)

	var int64Source int64

	err := binary.Read(bytes.NewReader(randomKey), binary.BigEndian, &int64Source)

	if err != nil {
		return nil, err
	}

	randSource := mathrand.New(mathrand.NewSource(int64Source))

	tasksSelector, err := random.NewDiscreteRandomVariable(randSource, weights)

	if errors.Is(err, random.ErrEmptyWeightsSlice) {
		return nil, app.ErrNotEnoughPhrasesInLesson
	}

	if err != nil {
		return nil, err
	}

	return &Lesson{
		phrases:         phrasesWithStatistics,
		randSource:      randSource,
		tasksProperties: tasksProperties,
		tasksSelector:   tasksSelector,
	}, nil
}

// Contains the logick of prioritizing tasks for their right order in lesson
// and more productive learning.
func calculateWeightOfTask(learningStatistics *PhraseLearningStatistics, kindOfTask kindOfTask, taskInverted bool) float64 {
	OneOptionChoiceTasksPassed := learningStatistics.CountGuessedOOS +
		learningStatistics.CountFailedOOS +
		learningStatistics.CountGuessedOOSInverted +
		learningStatistics.CountFailedOOSInverted

	OneOptionChoiceTasksPassedSuccessfully := learningStatistics.CountGuessedOOS +
		learningStatistics.CountGuessedOOSInverted

	//First 3 tasks are always translation from foreigh language
	//to known one by selecting the right card.
	if OneOptionChoiceTasksPassedSuccessfully < 3 {
		if kindOfTask == kindOfTaskChooseOneOption && !taskInverted {
			return 1
		}

		return 0
	}

	//Than mixed mode of cards: to foregn and to known language.
	if OneOptionChoiceTasksPassedSuccessfully < 7 {
		if kindOfTask == kindOfTaskChooseOneOption {
			//2 tasks are available for 1 phrase.
			//Weight 1.0 will cause that this phrase will become more prioritized
			//than other (sum of all tasks' weights for one phrase should be always 1.0).
			return 0.5
		}

		return 0
	}

	//7-10-th tasks by concrete phrase always will be translation
	//from known language to foreign by cards.
	if OneOptionChoiceTasksPassedSuccessfully < 10 {
		if kindOfTask == kindOfTaskChooseOneOption && taskInverted {
			return 1
		}

		return 0
	}

	if float64(OneOptionChoiceTasksPassedSuccessfully)/float64(OneOptionChoiceTasksPassed) < 0.7 {
		if kindOfTask == kindOfTaskChooseOneOption || (kindOfTask == kindOfTaskTranslateManually && !taskInverted) {
			//3 tasks are available for 1 phrase.
			//Weight 1.0 will cause that this phrase will become more prioritized
			//than other (sum of all tasks' weights for one phrase should be always 1.0).
			return float64(1) / float64(3)
		}

		return 0
	}

	TranslateManuallyTasksPassed := learningStatistics.CountAnsweredTM +
		learningStatistics.CountFailedTM +
		learningStatistics.CountAnsweredTMInverted +
		learningStatistics.CountFailedTMInverted

	TranslateManuallyTasksPassedSuccessfully := learningStatistics.CountAnsweredTM +
		learningStatistics.CountAnsweredTMInverted

	//When the phrase is complitely learned, we need to
	//make it less prioritized (to improve learning of other).
	//Don't set this weight to zero! It will cause
	//repetition of one phrase again and again when all the phrases
	//are learned.
	if TranslateManuallyTasksPassed > 10 && float64(TranslateManuallyTasksPassedSuccessfully)/float64(TranslateManuallyTasksPassed) > 0.9 {
		return 0.1
	}

	//All further learning (before 10 attempts with more than 90% probability of right answer)
	//will contain only manual translation.
	if kindOfTask == kindOfTaskTranslateManually {
		return 0.5
	}

	return 0
}

// Changes weights of all the tasks connected with phrase.
func (l *Lesson) setWeightsToTasks(pwsati *phraseWithStatisticsAndTasksIndexes, _ kindOfTask, _ bool) {
	l.tasksSelector.SetWeight(
		pwsati.IndexOfChooseRightOptionTask,
		calculateWeightOfTask(
			&pwsati.LearningStatistics,
			kindOfTaskChooseOneOption,
			false,
		),
	)

	l.tasksSelector.SetWeight(
		pwsati.IndexOfChooseRightOptionInvertedTask,
		calculateWeightOfTask(
			&pwsati.LearningStatistics,
			kindOfTaskChooseOneOption,
			true,
		),
	)

	l.tasksSelector.SetWeight(
		pwsati.IndexOfTranslateManuallyTask,
		calculateWeightOfTask(
			&pwsati.LearningStatistics,
			kindOfTaskTranslateManually,
			false,
		),
	)

	l.tasksSelector.SetWeight(
		pwsati.IndexOfTranslateManuallyInvertedTask,
		calculateWeightOfTask(
			&pwsati.LearningStatistics,
			kindOfTaskTranslateManually,
			true,
		),
	)
}

// Returns a slice of random unique indexes for slice l.phrases.
// Ignores tasks weights and statistics.
func (l *Lesson) getRandomPhrasesIndexes(n int) ([]int, error) {
	if n > len(l.phrases) {
		return nil, app.ErrNotEnoughPhrasesInLesson
	}

	indexes := make([]int, len(l.phrases))

	for i := range l.phrases {
		indexes[i] = i
	}

	l.randSource.Shuffle(
		len(l.phrases),
		func(i, j int) {
			indexes[i], indexes[j] = indexes[j], indexes[i]
		},
	)

	return indexes[:n], nil
}

// Gathers all the statistics and changes tasks' weights.
func (l *Lesson) taskSolved(task app.PhraseLearningTask, success bool) {
	var (
		phraseIndex int
		pwsati      *phraseWithStatisticsAndTasksIndexes
		kindOfTask  kindOfTask
	)

	switch t := task.(type) {
	case *oneOptionChoiceTask:
		kindOfTask = kindOfTaskChooseOneOption
		phraseIndex = t.PhraseIndex
		pwsati = &l.phrases[t.PhraseIndex]

		ls := &pwsati.LearningStatistics

		if t.IsInverted && success {
			ls.CountGuessedOOSInverted++
		} else if t.IsInverted && !success {
			ls.CountFailedOOSInverted++
		} else if !t.IsInverted && success {
			ls.CountGuessedOOS++
		} else if !t.IsInverted && !success {
			ls.CountFailedOOS++
		}
	case *tranclateManuallyTask:
		kindOfTask = kindOfTaskTranslateManually
		phraseIndex = t.PhraseIndex
		pwsati = &l.phrases[t.PhraseIndex]

		ls := &pwsati.LearningStatistics

		if t.IsInverted && success {
			ls.CountAnsweredTMInverted++
		} else if t.IsInverted && !success {
			ls.CountFailedTMInverted++
		} else if !t.IsInverted && success {
			ls.CountAnsweredTM++
		} else if !t.IsInverted && !success {
			ls.CountFailedTM++
		}
	}

	l.setWeightsToTasks(pwsati, kindOfTask, success)

	l.updateLastPhrasesWeights(phraseIndex)
}

func (l *Lesson) updateLastPhrasesWeights(updateStoredWeightsForPhrase int) {
	var (
		newWeight      float64
		phraseTaskData *weightWithIndex
	)

	for i, phraseData := range l.lastPhrasesToNotRepeat {
		if phraseData != nil {
			for j := range len(phraseData.Tasks) {
				phraseTaskData = &phraseData.Tasks[j]

				//If we are updating weights after
				//recalculation of tasks' weights, we have to save
				//updated weights (to not overwrite them when weights will be recovered).
				if updateStoredWeightsForPhrase == phraseData.PhraseIndex {
					phraseTaskData.Weight = l.tasksSelector.GetWeight(phraseTaskData.Index)
				}

				if i == len(l.lastPhrasesToNotRepeat)-1 {
					newWeight = 0
				} else if i == 0 {
					//Recovers tasks weight.
					newWeight = phraseTaskData.Weight
				} else {
					newWeight = phraseTaskData.Weight / float64(i+1)
				}

				l.tasksSelector.SetWeight(phraseTaskData.Index, newWeight)
			}
		}
	}
}

// Updates array lastPhrasesToNotRepeat and decreases weights of tasks connected
// with phrases from lastPhrasesToNotRepeat.
func (l *Lesson) updateLastPhrases(currentPhraseIndex int) {
	var newFirst *phraseWithTasksWeights

	//If this phrase already appeared during last 4 tasks
	//we will move in to the end of lastPhrasesToNotRepeat array.
	for i := range len(l.lastPhrasesToNotRepeat) {
		if l.lastPhrasesToNotRepeat[i] != nil && l.lastPhrasesToNotRepeat[i].PhraseIndex == currentPhraseIndex {
			newFirst = l.lastPhrasesToNotRepeat[i]

			l.lastPhrasesToNotRepeat[i] = nil

			break
		}
	}

	if newFirst == nil {
		newFirst = &phraseWithTasksWeights{
			PhraseIndex: currentPhraseIndex,
			Tasks:       make([]weightWithIndex, 0, 4),
		}

		for i, task := range l.tasksProperties {
			if task.PhraseIndex != currentPhraseIndex {
				continue
			}

			newFirst.Tasks = append(
				newFirst.Tasks,
				weightWithIndex{
					Index:  i,
					Weight: l.tasksSelector.GetWeight(i),
				},
			)
		}
	}

	//Moves array from end to begin (first phrase tasks' weights have already been recovered
	//by previous updateLastPhrases call).
	for i := range len(l.lastPhrasesToNotRepeat) - 1 {
		l.lastPhrasesToNotRepeat[i] = l.lastPhrasesToNotRepeat[i+1]
	}

	l.lastPhrasesToNotRepeat[len(l.lastPhrasesToNotRepeat)-1] = newFirst

	l.updateLastPhrasesWeights(-1)
}

// Returns the next task. Can be called before check of previous task.
// Avoids too often repetition of phrases.
func (l *Lesson) Next(ctx context.Context) (app.PhraseLearningTask, error) {
	var (
		taskProperties = l.tasksProperties[l.tasksSelector.Get()]
		res            app.PhraseLearningTask
	)

	//In case when task requested before the solution of previous one,
	//we need to change its priority here to avoid repetition without gap.
	defer l.updateLastPhrases(taskProperties.PhraseIndex)

	switch taskProperties.KnidOfTask {
	case kindOfTaskChooseOneOption:
		optionsCount := 8

		phrasesIndexes, err := l.getRandomPhrasesIndexes(optionsCount)

		if err != nil {
			return nil, err
		}

		var (
			options     = make([]string, 0, optionsCount)
			toTranslate string
			right       = -1
		)

		for i, index := range phrasesIndexes {
			if index == taskProperties.PhraseIndex {
				right = i

				break
			}
		}

		if right == -1 {
			right = l.randSource.Intn(optionsCount)

			phrasesIndexes[right] = taskProperties.PhraseIndex
		}

		for i, index := range phrasesIndexes {
			toAdd := l.phrases[index].Phrase

			if taskProperties.Inverted {
				toAdd.Invert()
			}

			if i == right {
				toTranslate = toAdd.Phrase
			}

			options = append(options, toAdd.Translation)
		}

		res = &oneOptionChoiceTask{
			PhraseToTranslate: toTranslate,
			AvailableOptions:  options,
			IsInverted:        taskProperties.Inverted,
			RightAnswer:       right,
			PhraseIndex:       phrasesIndexes[right],
			Solved:            l.taskSolved,
		}

	case kindOfTaskTranslateManually:
		//In case of kindOfTaskChooseOneOption this check implemented
		//in the getRandomPhrasesIndexes().
		if len(l.phrases) <= 0 {
			return nil, app.ErrNotEnoughPhrasesInLesson
		}

		phraseIndex := taskProperties.PhraseIndex
		taskPhrase := l.phrases[phraseIndex].Phrase

		if taskProperties.Inverted {
			taskPhrase.Invert()
		}

		res = &tranclateManuallyTask{
			PhraseToTranslate: taskPhrase,
			IsInverted:        taskProperties.Inverted,
			PhraseIndex:       phraseIndex,
			Solved:            l.taskSolved,
		}
	}

	return res, nil
}

func (l *Lesson) GetProgress() []PhraseWithLearningStatistics {
	res := make([]PhraseWithLearningStatistics, len(l.phrases))

	for i, phraseData := range l.phrases {
		res[i] = PhraseWithLearningStatistics{
			Phrase:             phraseData.Phrase,
			LearningStatistics: phraseData.LearningStatistics,
		}
	}

	return res
}
