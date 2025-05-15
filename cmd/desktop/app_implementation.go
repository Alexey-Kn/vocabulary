package main

import (
	"context"
	"vocabulary/internal/app"
	"vocabulary/internal/app/advanced"
	"vocabulary/internal/storage"
	"vocabulary/internal/ui"

	"slices"

	"github.com/xuri/excelize/v2"
)

type loadAllFile struct {
	excelFile    *excelize.File
	currentPath  string
	sheets       []string
	currentSheet string

	prevLesson         *advanced.Lesson
	prevLessonFilePath string
	prevLessonSheet    string
	storage            *storage.File
}

var _ ui.Application = (*loadAllFile)(nil)

func (ai *loadAllFile) close() {
	if ai.excelFile != nil {
		ai.excelFile.Close()

		ai.currentPath = ""
		ai.sheets = []string{}

		ai.excelFile = nil
	}
}

func (ai *loadAllFile) exit() {
	ai.saveProgressOfPrevLesson(nil, "", "")

	ai.close()
}

func (ai *loadAllFile) saveProgressOfPrevLesson(currentLesson *advanced.Lesson, cueerntLessonFilePath, currentLessonSheet string) {
	if ai.prevLesson != nil {
		phrasesLearningStatistics := ai.prevLesson.GetProgress()

		toStore := make(map[string]advanced.PhraseLearningStatistics, len(phrasesLearningStatistics))

		for _, phraseWithStats := range phrasesLearningStatistics {
			if !phraseWithStats.LearningStatistics.IsEmpty() {
				toStore[phraseWithStats.Phrase.Phrase] = phraseWithStats.LearningStatistics
			}
		}

		ai.storage.SaveLessonProgress(context.Background(), ai.prevLessonFilePath, ai.prevLessonSheet, toStore)
	}

	ai.prevLesson = currentLesson
	ai.prevLessonFilePath = cueerntLessonFilePath
	ai.prevLessonSheet = currentLessonSheet
}

func (ai *loadAllFile) OpenLast() error {
	path, sheet, err := ai.storage.LoadLastOpen(context.Background())

	if err != nil {
		return err
	}

	ai.OpenFile(path)

	ai.ChooseTopic(sheet)

	return nil
}

func (ai *loadAllFile) OpenFile(path string) bool {
	ai.close()

	file, err := excelize.OpenFile(path)

	if err != nil {
		return false
	}

	ai.excelFile = file
	ai.currentPath = path
	ai.sheets = file.GetSheetList()

	sheetFound := slices.Contains(ai.sheets, ai.currentSheet)

	if !sheetFound {
		ai.currentSheet = ai.sheets[0]
	}

	return true
}

func (ai *loadAllFile) FilePath() string {
	return ai.currentPath
}

func (ai *loadAllFile) AvailableTopics() []string {
	return ai.sheets
}

func (ai *loadAllFile) Topic() string {
	return ai.currentSheet
}

func (ai *loadAllFile) ChooseTopic(s string) {
	ai.currentSheet = s
}

func (ai *loadAllFile) ProgressRecoveryIsAvailable() bool {
	ai.saveProgressOfPrevLesson(nil, "", "")

	return ai.storage.SavedProgressAvailable(context.Background(), ai.currentPath, ai.currentSheet)
}

func (ai *loadAllFile) BeginLesson(recoverProgress bool) (app.Lesson, error) {
	rows, err := ai.excelFile.Rows(ai.currentSheet)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	ai.saveProgressOfPrevLesson(nil, "", "")

	var (
		phrases                  []advanced.PhraseWithLearningStatistics
		storedStatisticsByPhrase map[string]advanced.PhraseLearningStatistics
	)

	if recoverProgress {
		storedStatisticsByPhrase, err = ai.storage.LoadLessonProgress(context.Background(), ai.currentPath, ai.currentSheet)

		if err != nil {
			return nil, err
		}

		phrases = make([]advanced.PhraseWithLearningStatistics, 0, len(storedStatisticsByPhrase))
	} else {
		phrases = []advanced.PhraseWithLearningStatistics{}
	}

	for rows.Next() {
		cols, err := rows.Columns()

		if err != nil {
			return nil, err
		}

		if len(cols) < 2 {
			continue
		}

		phrase := cols[0]
		translation := cols[1]

		learningStatistics := advanced.PhraseLearningStatistics{}

		if recoverProgress {
			storedStatisticsForThisPhrase, found := storedStatisticsByPhrase[phrase]

			if found {
				learningStatistics = storedStatisticsForThisPhrase
			}
		}

		phrases = append(
			phrases,
			advanced.PhraseWithLearningStatistics{
				Phrase: app.PhraseWithTranslation{
					Phrase:      phrase,
					Translation: translation,
				},
				LearningStatistics: learningStatistics,
			},
		)
	}

	res, err := advanced.NewWithProgress(phrases)

	if err == nil {
		ai.storage.SaveLastOpen(context.Background(), ai.currentPath, ai.currentSheet)

		ai.saveProgressOfPrevLesson(res, ai.currentPath, ai.currentSheet)
	}

	return res, err
}
