package main

import "time"

const (
	TIME_TO_STORE_LESSONS_PROGRESS      = time.Hour * 24 * 30 * 3
	MAX_LESSONS_COUNT_TO_STORE_PROGRESS = 5000
	STORAGE_FILE_PATH                   = "./storage"
)
