package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"vocabulary/internal/app"
	"vocabulary/internal/app/advanced"

	_ "github.com/mattn/go-sqlite3"
)

var ErrWasNotSaved = errors.New("wasn't saved")

type File struct {
	db *sql.DB
}

// Opens or creates an sqlite database by given file path.
// If filePath argument doesn't include extention, it will be added.
func Open(ctx context.Context, filePath string) (*File, error) {
	if !strings.HasSuffix(filePath, FILE_EXTENTION) {
		filePath += FILE_EXTENTION
	}

	db, err := sql.Open("sqlite3", filePath)

	if err != nil {
		return nil, err
	}

	initRequestText := `
		CREATE TABLE IF NOT EXISTS EXCEL_LESSONS
		(
			ID INTEGER PRIMARY KEY AUTOINCREMENT,
			DATE_UTC TEXT NOT NULL,
			FILE_PATH TEXT NOT NULL,
			FILE_SHEET TEXT NOT NULL,
			MODE INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS LESSONS_PROGRESS
		(
			EXCEL_LESSON INTEGER NOT NULL,
			PHRASE TEXT NOT NULL,
			COUNT_GUESSED_OOS INTEGER NOT NULL,
			COUNT_FAILED_OOS INTEGER NOT NULL,
			COUNT_ANSWERED_TM INTEGER NOT NULL,
			COUNT_FAILED_TM INTEGER NOT NULL,
			COUNT_GUESSED_OOS_INVERTED INTEGER NOT NULL,
			COUNT_FAILED_OOS_INVERTED INTEGER NOT NULL,
			COUNT_ANSWERED_TM_INVERTED INTEGER NOT NULL,
			COUNT_FAILED_TM_INVERTED INTEGER NOT NULL,
			FOREIGN KEY (EXCEL_LESSON) REFERENCES EXCEL_LESSONS(ID) ON DELETE CASCADE
		);
	`

	_, err = db.Exec(initRequestText)

	if err != nil {
		db.Close()

		return nil, err
	}

	res := &File{
		db: db,
	}

	return res, nil
}

func (s *File) SaveLastOpen(ctx context.Context, excelFilePath, sheet string, mode app.LessonMode) error {
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	err = s.updateExcelLessonDateOrAddExcelLesson(ctx, tx, excelFilePath, sheet, mode)

	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	return tx.Commit()
}

func (s *File) updateExcelLessonDateOrAddExcelLesson(ctx context.Context, tx *sql.Tx, excelFilePath, sheet string, mode app.LessonMode) error {
	timeUTC := time.Now().UTC().Format(SQLITE_TIME_FORMAT)

	requestText := `
		UPDATE EXCEL_LESSONS
		SET DATE_UTC = ?, MODE = ?
		WHERE FILE_PATH = ?
		AND FILE_SHEET = ?
	`

	requestRes, err := tx.ExecContext(ctx, requestText, timeUTC, mode, excelFilePath, sheet)

	if err != nil {
		return err
	}

	rowsAffected, err := requestRes.RowsAffected()

	if err != nil {
		return err
	}

	if rowsAffected <= 0 {
		requestText = `
			INSERT INTO EXCEL_LESSONS (DATE_UTC, FILE_PATH, FILE_SHEET, MODE) VALUES
			(?, ?, ?, ?)
		`

		_, err = tx.ExecContext(ctx, requestText, timeUTC, excelFilePath, sheet, mode)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *File) LoadLastOpen(ctx context.Context) (excelFilePath, sheet string, mode app.LessonMode, err error) {
	requestText := `
		SELECT FILE_PATH, FILE_SHEET, MODE
		FROM EXCEL_LESSONS
		ORDER BY DATE_UTC DESC
		LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, requestText)

	err = row.Scan(&excelFilePath, &sheet, &mode)

	if errors.Is(err, sql.ErrNoRows) {
		return "", "", 0, ErrWasNotSaved
	}

	return excelFilePath, sheet, mode, err
}

func (s *File) SavedProgressAvailable(ctx context.Context, excelFilePath, sheet string) bool {
	requestText := `
		SELECT COUNT(*) > 0
		FROM EXCEL_LESSONS JOIN LESSONS_PROGRESS
			ON EXCEL_LESSONS.ID = LESSONS_PROGRESS.EXCEL_LESSON
		WHERE
			EXCEL_LESSONS.FILE_PATH = ? AND EXCEL_LESSONS.FILE_SHEET = ?
	`

	row := s.db.QueryRowContext(ctx, requestText, excelFilePath, sheet)

	res := false

	err := row.Scan(&res)

	if err != nil {
		return false
	}

	return res
}

func (s *File) SaveLessonProgress(
	ctx context.Context,
	excelFilePath string,
	sheet string,
	statisticsByPhrase map[string]advanced.PhraseLearningStatistics,
) error {
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	requestText := `
		DELETE FROM LESSONS_PROGRESS
		WHERE EXCEL_LESSON IN (
			SELECT ID
			FROM EXCEL_LESSONS
			WHERE FILE_PATH = ? AND FILE_SHEET = ?
		)
	`

	_, err = tx.ExecContext(ctx, requestText, excelFilePath, sheet)

	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	err = s.updateExcelLessonDateOrAddExcelLesson(ctx, tx, excelFilePath, sheet, app.LessonModeLern)

	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	requestText = `
		SELECT ID
		FROM EXCEL_LESSONS
		WHERE FILE_PATH = ? AND FILE_SHEET = ?
	`

	row := tx.QueryRowContext(ctx, requestText, excelFilePath, sheet)

	var lessonID int64

	err = row.Scan(&lessonID)

	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	requestText = `
		INSERT INTO LESSONS_PROGRESS
		(
			EXCEL_LESSON,
			PHRASE,
			COUNT_GUESSED_OOS,
			COUNT_FAILED_OOS,
			COUNT_ANSWERED_TM,
			COUNT_FAILED_TM,
			COUNT_GUESSED_OOS_INVERTED,
			COUNT_FAILED_OOS_INVERTED,
			COUNT_ANSWERED_TM_INVERTED,
			COUNT_FAILED_TM_INVERTED
		)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	preparedRequest, err := tx.PrepareContext(ctx, requestText)

	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	for phrase, stats := range statisticsByPhrase {
		_, err = preparedRequest.ExecContext(
			ctx,
			lessonID,
			phrase,
			stats.CountGuessedOOS,
			stats.CountFailedOOS,
			stats.CountAnsweredTM,
			stats.CountFailedTM,
			stats.CountGuessedOOSInverted,
			stats.CountFailedOOSInverted,
			stats.CountAnsweredTMInverted,
			stats.CountFailedTMInverted,
		)

		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
	}

	return tx.Commit()
}

func (s *File) LoadLessonProgress(
	ctx context.Context,
	excelFilePath,
	sheet string,
) (
	statisticsByPhrase map[string]advanced.PhraseLearningStatistics,
	err error,
) {
	requestText := `
		SELECT
			LESSONS_PROGRESS.PHRASE,
			LESSONS_PROGRESS.COUNT_GUESSED_OOS,
			LESSONS_PROGRESS.COUNT_FAILED_OOS,
			LESSONS_PROGRESS.COUNT_ANSWERED_TM,
			LESSONS_PROGRESS.COUNT_FAILED_TM,
			LESSONS_PROGRESS.COUNT_GUESSED_OOS_INVERTED,
			LESSONS_PROGRESS.COUNT_FAILED_OOS_INVERTED,
			LESSONS_PROGRESS.COUNT_ANSWERED_TM_INVERTED,
			LESSONS_PROGRESS.COUNT_FAILED_TM_INVERTED
		FROM EXCEL_LESSONS JOIN LESSONS_PROGRESS
			ON EXCEL_LESSONS.ID = LESSONS_PROGRESS.EXCEL_LESSON
		WHERE
			EXCEL_LESSONS.FILE_PATH = ? AND EXCEL_LESSONS.FILE_SHEET = ?
	`

	query, err := s.db.QueryContext(ctx, requestText, excelFilePath, sheet)

	if err != nil {
		return nil, err
	}

	defer query.Close()

	var (
		res    = map[string]advanced.PhraseLearningStatistics{}
		stats  advanced.PhraseLearningStatistics
		phrase string
	)

	for query.Next() {
		err = query.Scan(
			&phrase,
			&stats.CountGuessedOOS,
			&stats.CountFailedOOS,
			&stats.CountAnsweredTM,
			&stats.CountFailedTM,
			&stats.CountGuessedOOSInverted,
			&stats.CountFailedOOSInverted,
			&stats.CountAnsweredTMInverted,
			&stats.CountFailedTMInverted,
		)

		if err != nil {
			return nil, err
		}

		res[phrase] = stats
	}

	if query.Err() != nil {
		return nil, query.Err()
	}

	if len(res) <= 0 {
		return nil, ErrWasNotSaved
	}

	return res, nil
}

// Removes all the data associated with lessons which were used earlier than excelLessonsHistoryPeriodBeginning.
// Removes lesson if only it's number (by the order of decreasing last usage date) is bigger than maxLessonsCount.
// Uses FIFO discipline.
func (s *File) EraseOutdatedData(ctx context.Context, maxLessonsCount uint32, excelLessonsHistoryPeriodBeginning time.Time) error {
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	periodInSQLiteFormat := excelLessonsHistoryPeriodBeginning.Format(SQLITE_TIME_FORMAT)

	requestText := `
		WITH
			NUMBERED AS (
				SELECT ID, DATE_UTC, ROW_NUMBER() OVER (ORDER BY DATE_UTC DESC) AS RN
				FROM EXCEL_LESSONS
			),

			TO_DELETE AS (
				SELECT ID
				FROM NUMBERED
				WHERE RN > ? AND DATE_UTC < ?
			)

		DELETE FROM LESSONS_PROGRESS
		WHERE EXCEL_LESSON IN TO_DELETE
	`

	_, err = tx.ExecContext(ctx, requestText, maxLessonsCount, periodInSQLiteFormat)

	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	requestText = `
		WITH
			NUMBERED AS (
				SELECT ID, DATE_UTC, ROW_NUMBER() OVER (ORDER BY DATE_UTC DESC) AS RN
				FROM EXCEL_LESSONS
			),

			TO_DELETE AS (
				SELECT ID
				FROM NUMBERED
				WHERE RN > ? AND DATE_UTC < ?
			)

		DELETE FROM EXCEL_LESSONS
		WHERE ID IN TO_DELETE
	`

	_, err = tx.ExecContext(ctx, requestText, maxLessonsCount, periodInSQLiteFormat)

	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	return tx.Commit()
}

func (s *File) Close() error {
	return s.db.Close()
}
